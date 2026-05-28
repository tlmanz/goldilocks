package dashboard

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/summary"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

// NamespaceListItem is the per-namespace data shape rendered into each
// namespace card on the listing page. Stats fields are zero when the
// namespace is labelled for Goldilocks but doesn't yet have any VPAs.
type NamespaceListItem struct {
	Name                   string
	WorkloadCount          int
	ContainerCount         int
	NeedsAttention         int
	LowConfidence          int
	OverCount              int
	UnderCount             int
	MissingCount           int
	EqualCount             int
	SavingsScore           int64
	LastRecommendationUnix int64
}

// NamespaceList replies with the rendered namespace list. When the summary
// cache is supplied we reuse it (so a recent dashboard load makes this page
// effectively free) and enrich each namespace card with workload counts,
// status, and savings stats.
func NamespaceList(opts Options, cache *summaryCache) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var listOptions v1.ListOptions
		if opts.OnByDefault || opts.ShowAllVPAs {
			listOptions = v1.ListOptions{
				LabelSelector: fmt.Sprintf("%s!=false", utils.VpaEnabledLabel),
			}
		} else {
			listOptions = v1.ListOptions{
				LabelSelector: labels.Set(map[string]string{
					utils.VpaEnabledLabel: "true",
				}).String(),
			}
		}
		namespacesList, err := kube.GetInstance().Client.CoreV1().Namespaces().List(context.TODO(), listOptions)
		if err != nil {
			klog.Errorf("Error getting namespace list: %v", err)
			http.Error(w, "Error getting namespace list", http.StatusInternalServerError)
			return
		}

		tmpl, err := getTemplate("namespace_list", opts,
			"filter",
			"namespace_list",
		)
		if err != nil {
			klog.Errorf("Error getting template data: %v", err)
			http.Error(w, "Error getting template data", http.StatusInternalServerError)
			return
		}

		// Pull the cluster-wide summary through the shared cache so we can
		// enrich the listing without paying a second VPA list when the
		// dashboard was just loaded. Failures here are non-fatal — we still
		// render the (stats-less) list.
		var enrich summary.Summary
		if cache != nil {
			data, _, _, err := cache.Get("", "", "", func() (summary.Summary, error) {
				return getVPAData(opts, "", "", "")
			})
			if err != nil {
				klog.Warningf("namespace list: enrich via cache failed: %v", err)
			} else {
				enrich = data
			}
		}

		// only expose the needed data from Namespace — keep this consistent
		// with the original handler so we don't leak metadata.
		data := struct {
			Namespaces []NamespaceListItem
		}{}

		for _, ns := range namespacesList.Items {
			item := NamespaceListItem{Name: ns.Name}
			if nsSummary, ok := enrich.Namespaces[ns.Name]; ok {
				item.WorkloadCount = len(nsSummary.Workloads)
				containerCount := 0
				for _, w := range nsSummary.Workloads {
					containerCount += len(w.Containers)
				}
				item.ContainerCount = containerCount
				item.NeedsAttention = nsSummary.NeedsAttention
				item.LowConfidence = nsSummary.LowConfidence
				item.OverCount = nsSummary.OverCount
				item.UnderCount = nsSummary.UnderCount
				item.MissingCount = nsSummary.MissingCount
				item.EqualCount = nsSummary.EqualCount
				item.SavingsScore = nsSummary.SavingsScore
				item.LastRecommendationUnix = nsSummary.LastRecommendationUnix
			}
			data.Namespaces = append(data.Namespaces, item)
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}
