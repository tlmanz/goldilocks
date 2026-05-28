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
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/history"
)

// History serves a JSON time-series of one container's recorded snapshots.
// Returns 503 if no history store is configured.
//
// Query params:
//
//	ns        — namespace (required)
//	kind      — workload kind (required)
//	workload  — workload name (required)
//	container — container name (required)
//	hours     — lookback window in hours (default 24, capped at 720 = 30d)
func History(store *history.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			http.Error(w, "history is not enabled on this dashboard (pass --history-db)", http.StatusServiceUnavailable)
			return
		}

		q := r.URL.Query()
		namespace := q.Get("ns")
		kind := q.Get("kind")
		workload := q.Get("workload")
		container := q.Get("container")
		if namespace == "" || kind == "" || workload == "" || container == "" {
			http.Error(w, "ns, kind, workload, and container query params are required", http.StatusBadRequest)
			return
		}

		hours := 24
		if hStr := q.Get("hours"); hStr != "" {
			if n, err := strconv.Atoi(hStr); err == nil && n > 0 {
				if n > 720 {
					n = 720
				}
				hours = n
			}
		}
		since := time.Now().Add(-time.Duration(hours) * time.Hour)

		points, err := store.Series(r.Context(), namespace, kind, workload, container, since)
		if err != nil {
			klog.Errorf("history: query failed: %v", err)
			http.Error(w, "failed to read history", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=30")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"namespace":  namespace,
			"kind":       kind,
			"workload":   workload,
			"container":  container,
			"sinceUnix":  since.Unix(),
			"points":     points,
		}); err != nil {
			klog.Errorf("history: encode response: %v", err)
		}
	})
}
