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

package helpers

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func PrintResource(quant resource.Quantity) string {
	if quant.IsZero() {
		return "Not Set"
	}
	return quant.String()
}

func GetStatus(existing resource.Quantity, recommendation resource.Quantity, style string) string {
	if existing.IsZero() {
		switch style {
		case "text":
			return "error - not set"
		case "icon":
			return "fa-exclamation error"
		default:
			return ""
		}
	}

	comparison := existing.Cmp(recommendation)
	if comparison == 0 {
		switch style {
		case "text":
			return "equal"
		case "icon":
			return "fa-equals success"
		default:
			return ""
		}
	}
	if comparison < 0 {
		switch style {
		case "text":
			return "less than"
		case "icon":
			return "fa-less-than warning"
		default:
			return ""
		}
	}
	if comparison > 0 {
		switch style {
		case "text":
			return "greater than"
		case "icon":
			return "fa-greater-than warning"
		default:
			return ""
		}
	}
	return ""
}

func GetStatusRange(existing, lower, upper resource.Quantity, style string, resourceType string) string {
	if existing.IsZero() {
		switch style {
		case "text":
			return "error - not set"
		case "icon":
			return "fa-exclamation error"
		default:
			return ""
		}
	}

	comparisonLower := existing.Cmp(lower)
	comparisonUpper := existing.Cmp(upper)

	if comparisonLower < 0 {
		switch style {
		case "text":
			return "less than"
		case "icon":
			return "fa-less-than warning"
		}
	}

	if comparisonUpper > 0 {
		switch style {
		case "text":
			return "greater than"
		case "icon":
			return "fa-greater-than warning"
		}
	}

	switch resourceType {
	case "request":
		if comparisonLower == 0 {
			switch style {
			case "text":
				return "equal"
			case "icon":
				return "fa-equals success"
			}
		}
	case "limit":
		if comparisonUpper == 0 {
			switch style {
			case "text":
				return "equal"
			case "icon":
				return "fa-equals success"
			}
		}
	}

	switch style {
	case "text":
		return "not equal"
	case "icon":
		return "fa-exclamation error"
	}

	return ""
}

func ResourceName(name string) corev1.ResourceName {
	return corev1.ResourceName(name)
}

func GetUUID() string {
	return uuid.New().String()
}

func HasField(v any, name string) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	return rv.FieldByName(name).IsValid()
}

// ControllerIcon returns a FontAwesome icon class for a Kubernetes controller type.
func ControllerIcon(controllerType string) string {
	switch strings.ToLower(controllerType) {
	case "deployment":
		return "fa-rocket"
	case "statefulset":
		return "fa-database"
	case "daemonset":
		return "fa-server"
	case "job":
		return "fa-stopwatch"
	case "cronjob":
		return "fa-calendar"
	case "replicaset":
		return "fa-clone"
	case "pod":
		return "fa-cube"
	}
	return "fa-cubes"
}

var slugifyNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts an arbitrary string into a URL-safe slug suitable for HTML id attributes.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = slugifyNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// DiffPercent returns the magnitude of (current - target) / target as a percentage clamped to 0..200.
// Used by the diff-bar rendering on each compTable row.
func DiffPercent(current, target resource.Quantity) int {
	if target.IsZero() {
		if current.IsZero() {
			return 0
		}
		return 100
	}
	cur := current.MilliValue()
	tgt := target.MilliValue()
	if tgt == 0 {
		return 0
	}
	diff := cur - tgt
	if diff < 0 {
		diff = -diff
	}
	pct := int((diff * 100) / tgt)
	if pct > 200 {
		pct = 200
	}
	return pct
}

// DiffDirection returns "over", "under", or "equal" for a current vs target comparison.
func DiffDirection(current, target resource.Quantity) string {
	if current.IsZero() {
		return "missing"
	}
	cmp := current.Cmp(target)
	if cmp > 0 {
		return "over"
	}
	if cmp < 0 {
		return "under"
	}
	return "equal"
}

// ContainerState returns the dominant deviation state for a container based on
// the current CPU/memory requests vs the VPA Guaranteed target. The returned
// values align with the data-state attribute used for click-to-filter on the
// summary cards.
//
// Returns one of: "missing" (no request set), "over" (any resource above target),
// "under" (any resource below target without being over), or "equal".
func ContainerState(requests, target corev1.ResourceList) string {
	cpuName := corev1.ResourceName("cpu")
	memName := corev1.ResourceName("memory")

	cpuReq := requests[cpuName]
	memReq := requests[memName]
	cpuTarget := target[cpuName]
	memTarget := target[memName]

	if cpuReq.IsZero() || memReq.IsZero() {
		return "missing"
	}

	cpuDiff := cpuReq.MilliValue() - cpuTarget.MilliValue()
	memDiff := memReq.Value() - memTarget.Value()

	if cpuDiff > 0 || memDiff > 0 {
		return "over"
	}
	if cpuDiff < 0 || memDiff < 0 {
		return "under"
	}
	return "equal"
}

// Dict builds a map from alternating key/value template arguments so a template
// can pass multiple named values to a sub-template invocation.
func Dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires an even number of arguments")
	}
	d := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %T at position %d", values[i], i)
		}
		d[key] = values[i+1]
	}
	return d, nil
}
