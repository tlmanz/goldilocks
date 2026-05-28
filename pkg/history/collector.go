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

package history

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/summary"
)

// Collector periodically walks the Goldilocks summary and writes one snapshot
// row per container into the Store.
type Collector struct {
	Store      *Store
	Interval   time.Duration
	Retention  time.Duration
	NewSummary func() (summary.Summary, error)
}

// Run blocks until ctx is cancelled. It writes a snapshot every Interval and
// prunes anything older than Retention on every tick.
func (c *Collector) Run(ctx context.Context) {
	if c.Store == nil || c.NewSummary == nil {
		klog.Warning("history collector: missing Store or NewSummary, exiting")
		return
	}
	if c.Interval <= 0 {
		c.Interval = 5 * time.Minute
	}

	klog.Infof("history collector: starting (interval=%s retention=%s db=%s)", c.Interval, c.Retention, c.Store.Path())

	// Tick immediately so the first snapshot lands without waiting Interval.
	c.tick(ctx)

	t := time.NewTicker(c.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			klog.Info("history collector: context cancelled, exiting")
			return
		case <-t.C:
			c.tick(ctx)
		}
	}
}

func (c *Collector) tick(ctx context.Context) {
	now := time.Now()
	s, err := c.NewSummary()
	if err != nil {
		klog.Errorf("history collector: build summary: %v", err)
		return
	}
	snapshots := snapshotsFromSummary(now, s)
	if err := c.Store.Write(ctx, snapshots); err != nil {
		klog.Errorf("history collector: write %d snapshots: %v", len(snapshots), err)
		return
	}
	klog.V(2).Infof("history collector: wrote %d snapshots", len(snapshots))

	if c.Retention > 0 {
		cutoff := now.Add(-c.Retention)
		if n, err := c.Store.Prune(ctx, cutoff); err != nil {
			klog.Errorf("history collector: prune: %v", err)
		} else if n > 0 {
			klog.V(2).Infof("history collector: pruned %d rows older than %s", n, cutoff)
		}
	}
}

// snapshotsFromSummary flattens a Goldilocks summary into rows ready for Write.
func snapshotsFromSummary(now time.Time, s summary.Summary) []Snapshot {
	out := make([]Snapshot, 0, 64)
	cpu := corev1.ResourceName("cpu")
	mem := corev1.ResourceName("memory")

	for _, ns := range s.Namespaces {
		for _, w := range ns.Workloads {
			for _, c := range w.Containers {
				sn := Snapshot{
					Timestamp:    now,
					Namespace:    ns.Namespace,
					WorkloadKind: w.ControllerType,
					WorkloadName: w.ControllerName,
					Container:    c.ContainerName,
				}
				if v, ok := c.Requests[cpu]; ok && !v.IsZero() {
					m := v.MilliValue()
					sn.CPURequestM = &m
				}
				if v, ok := c.Limits[cpu]; ok && !v.IsZero() {
					m := v.MilliValue()
					sn.CPULimitM = &m
				}
				if v, ok := c.Requests[mem]; ok && !v.IsZero() {
					b := v.Value()
					sn.MemRequestB = &b
				}
				if v, ok := c.Limits[mem]; ok && !v.IsZero() {
					b := v.Value()
					sn.MemLimitB = &b
				}
				if v, ok := c.Target[cpu]; ok && !v.IsZero() {
					m := v.MilliValue()
					sn.CPUTargetM = &m
				}
				if v, ok := c.Target[mem]; ok && !v.IsZero() {
					b := v.Value()
					sn.MemTargetB = &b
				}
				if v, ok := c.LowerBound[cpu]; ok && !v.IsZero() {
					m := v.MilliValue()
					sn.CPULowerM = &m
				}
				if v, ok := c.UpperBound[cpu]; ok && !v.IsZero() {
					m := v.MilliValue()
					sn.CPUUpperM = &m
				}
				if v, ok := c.LowerBound[mem]; ok && !v.IsZero() {
					b := v.Value()
					sn.MemLowerB = &b
				}
				if v, ok := c.UpperBound[mem]; ok && !v.IsZero() {
					b := v.Value()
					sn.MemUpperB = &b
				}
				out = append(out, sn)
			}
		}
	}
	return out
}
