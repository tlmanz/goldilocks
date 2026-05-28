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
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/summary"
)

// summaryCache memoizes the summary + stats build for a short TTL so that
// repeated dashboard/api requests don't all rebuild from scratch.
//
// Both the Dashboard and API endpoints walk the entire VPA + workload set and
// produce a summary on every request — for clusters with many namespaces this
// is hundreds of milliseconds of work per request. Since VPA recommendations
// update on the order of minutes, a short TTL cache reclaims most of that.
type summaryCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[cacheKey]*cacheEntry
}

type cacheKey struct {
	namespace  string
	costPerCPU string
	costPerGB  string
}

type cacheEntry struct {
	expires time.Time
	data    summary.Summary
	stats   DashboardStats
	builtAt int64 // unix seconds when this entry was computed
}

// newSummaryCache returns a cache with the given TTL. A zero ttl disables
// caching entirely (every Get goes straight through to the builder).
func newSummaryCache(ttl time.Duration) *summaryCache {
	return &summaryCache{
		ttl:   ttl,
		items: map[cacheKey]*cacheEntry{},
	}
}

// Get returns a cached summary if fresh, otherwise builds via the provided
// loader, post-processes it (drop scaled-to-zero, populate per-namespace stats,
// compute dashboard stats), and caches the result.
func (c *summaryCache) Get(
	namespace, costPerCPU, costPerGB string,
	build func() (summary.Summary, error),
) (summary.Summary, DashboardStats, int64, error) {
	key := cacheKey{namespace, costPerCPU, costPerGB}

	if c.ttl > 0 {
		c.mu.Lock()
		entry, ok := c.items[key]
		c.mu.Unlock()
		if ok && time.Now().Before(entry.expires) {
			klog.V(4).Infof("summary cache hit ns=%q", namespace)
			return entry.data, entry.stats, entry.builtAt, nil
		}
	}

	klog.V(4).Infof("summary cache miss ns=%q", namespace)
	data, err := build()
	if err != nil {
		return summary.Summary{}, DashboardStats{}, 0, err
	}

	dropScaledToZero(&data)
	populatePerNamespaceStats(&data)
	stats := computeDashboardStats(data)
	builtAt := time.Now().Unix()

	if c.ttl > 0 {
		c.mu.Lock()
		c.items[key] = &cacheEntry{
			expires: time.Now().Add(c.ttl),
			data:    data,
			stats:   stats,
			builtAt: builtAt,
		}
		c.mu.Unlock()
	}

	return data, stats, builtAt, nil
}

// Invalidate clears all cached entries. Useful for tests; not currently
// exposed via a route, but available for future plumbing.
func (c *summaryCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[cacheKey]*cacheEntry{}
}
