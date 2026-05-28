// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dashboard

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/fairwindsops/goldilocks/pkg/summary"
)

// DashboardStats holds aggregated metrics computed from a VPA summary for
// rendering the top-of-dashboard insights bar.
type DashboardStats struct {
	NamespaceCount    int
	WorkloadCount     int
	ContainerCount    int
	OverProvisioned   int
	UnderProvisioned  int
	NotConfigured     int
	WellTuned         int
	CPUSavings        string
	MemorySavings     string
	DollarSavings     float64
	HasDollarSavings  bool
	HasCPUSavings     bool
	HasMemorySavings  bool
}

// computeDashboardStats walks the VPA summary and produces aggregate counts
// and potential-savings totals.
func computeDashboardStats(s summary.Summary) DashboardStats {
	stats := DashboardStats{}
	stats.NamespaceCount = len(s.Namespaces)

	cpuMillis := int64(0)
	memBytes := int64(0)
	cpuName := corev1.ResourceName("cpu")
	memName := corev1.ResourceName("memory")

	for _, ns := range s.Namespaces {
		stats.WorkloadCount += len(ns.Workloads)
		for _, w := range ns.Workloads {
			for _, c := range w.Containers {
				stats.ContainerCount++

				cpuReq := c.Requests[cpuName]
				memReq := c.Requests[memName]
				cpuTarget := c.Target[cpuName]
				memTarget := c.Target[memName]

				if cpuReq.IsZero() || memReq.IsZero() {
					stats.NotConfigured++
				}

				cpuDiff := cpuReq.MilliValue() - cpuTarget.MilliValue()
				memDiff := memReq.Value() - memTarget.Value()

				switch {
				case cpuDiff > 0 || memDiff > 0:
					stats.OverProvisioned++
				case cpuDiff < 0 || memDiff < 0:
					stats.UnderProvisioned++
				default:
					stats.WellTuned++
				}

				if cpuDiff > 0 {
					cpuMillis += cpuDiff
				}
				if memDiff > 0 {
					memBytes += memDiff
				}

				if c.GuaranteedCostInt < 0 {
					stats.DollarSavings += c.GuaranteedCost
				}
			}
		}
	}

	if cpuMillis > 0 {
		stats.CPUSavings = formatCPU(cpuMillis)
		stats.HasCPUSavings = true
	}
	if memBytes > 0 {
		stats.MemorySavings = formatMemory(memBytes)
		stats.HasMemorySavings = true
	}
	if stats.DollarSavings > 0 {
		stats.HasDollarSavings = true
	}

	return stats
}

func formatCPU(millicores int64) string {
	if millicores < 1000 {
		return fmt.Sprintf("%dm", millicores)
	}
	cores := float64(millicores) / 1000
	if cores >= 10 {
		return fmt.Sprintf("%.0f cores", cores)
	}
	return fmt.Sprintf("%.1f cores", cores)
}

func formatMemory(bytes int64) string {
	q := resource.NewQuantity(bytes, resource.BinarySI)
	return q.String()
}

// dropScaledToZero removes workloads whose desired replica count is explicitly
// zero, and removes namespaces that end up with no remaining workloads.
// Workload types where replica count is not applicable (Job, CronJob, etc.)
// are left untouched.
func dropScaledToZero(s *summary.Summary) {
	for nsName, ns := range s.Namespaces {
		for wName, w := range ns.Workloads {
			if w.Replicas != nil && *w.Replicas == 0 {
				delete(ns.Workloads, wName)
			}
		}
		if len(ns.Workloads) == 0 {
			delete(s.Namespaces, nsName)
		}
	}
}

// populatePerNamespaceStats walks the summary once and fills the per-namespace
// aggregate counters (NeedsAttention / LowConfidence / SavingsScore) used by
// the namespace card chrome and the "potential savings" sort.
func populatePerNamespaceStats(s *summary.Summary) {
	cpuName := corev1.ResourceName("cpu")
	memName := corev1.ResourceName("memory")

	for nsName, ns := range s.Namespaces {
		needs := 0
		lowConf := 0
		var savings int64
		over, under, missing, equal := 0, 0, 0, 0

		for _, w := range ns.Workloads {
			if w.LowConfidence {
				lowConf++
			}
			for _, c := range w.Containers {
				cpuReq := c.Requests[cpuName]
				memReq := c.Requests[memName]
				cpuTarget := c.Target[cpuName]
				memTarget := c.Target[memName]

				cpuDiff := cpuReq.MilliValue() - cpuTarget.MilliValue()
				memDiff := memReq.Value() - memTarget.Value()

				switch {
				case cpuReq.IsZero() || memReq.IsZero():
					needs++
					missing++
				case cpuDiff > 0 || memDiff > 0:
					needs++
					over++
					if cpuDiff > 0 {
						savings += cpuDiff
					}
					if memDiff > 0 {
						savings += memDiff / (1024 * 1024)
					}
				case cpuDiff < 0 || memDiff < 0:
					needs++
					under++
				default:
					equal++
				}
			}
		}

		ns.NeedsAttention = needs
		ns.LowConfidence = lowConf
		ns.SavingsScore = savings
		ns.OverCount = over
		ns.UnderCount = under
		ns.MissingCount = missing
		ns.EqualCount = equal
		s.Namespaces[nsName] = ns
	}
}
