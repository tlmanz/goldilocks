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

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/dashboard"
	"github.com/fairwindsops/goldilocks/pkg/history"
	"github.com/fairwindsops/goldilocks/pkg/summary"
)

var (
	serverPort         int
	showAllVPAs        bool
	basePath           string
	insightsHost       string
	enableCost         bool
	cacheTTL           time.Duration
	enableGzip         bool
	dashboardHistoryDB        string
	dashboardCollectHistory   bool
	dashboardHistoryInterval  time.Duration
	dashboardHistoryRetention time.Duration
)

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.PersistentFlags().IntVarP(&serverPort, "port", "p", 8080, "The port to serve the dashboard on.")
	dashboardCmd.PersistentFlags().StringVarP(&excludeContainers, "exclude-containers", "e", "", "Comma delimited list of containers to exclude from recommendations.")
	dashboardCmd.PersistentFlags().BoolVar(&onByDefault, "on-by-default", false, "Display every namespace that isn't explicitly excluded.")
	dashboardCmd.PersistentFlags().BoolVar(&showAllVPAs, "show-all", false, "Display every VPA, even if it isn't managed by Goldilocks")
	dashboardCmd.PersistentFlags().StringVar(&basePath, "base-path", "/", "Path on which the dashboard is served.")
	dashboardCmd.PersistentFlags().BoolVar(&enableCost, "enable-cost", true, "If set to false, the cost integration will be disabled on the dashboard.")
	dashboardCmd.PersistentFlags().StringVar(&insightsHost, "insights-host", "https://insights.fairwinds.com", "Insights host for retrieving optional cost data.")
	dashboardCmd.PersistentFlags().DurationVar(&cacheTTL, "cache-ttl", 30*time.Second, "How long to memoize the dashboard summary in memory. Set to 0 to disable caching.")
	dashboardCmd.PersistentFlags().BoolVar(&enableGzip, "enable-gzip", true, "Gzip-compress HTML and JSON responses.")
	dashboardCmd.PersistentFlags().StringVar(&dashboardHistoryDB, "history-db", "", "Path to the SQLite history database. When set, the dashboard exposes /api/history and renders trend sparklines.")
	dashboardCmd.PersistentFlags().BoolVar(&dashboardCollectHistory, "collect-history", true, "Run the history collector inside the dashboard Pod (writes to --history-db). Disable if a sidecar / separate controller is doing the writes.")
	dashboardCmd.PersistentFlags().DurationVar(&dashboardHistoryInterval, "history-interval", 5*time.Minute, "How often the in-process collector writes a snapshot.")
	dashboardCmd.PersistentFlags().DurationVar(&dashboardHistoryRetention, "history-retention", 168*time.Hour, "How long to keep snapshots. Zero disables pruning.")
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Run the goldilocks dashboard that will show recommendations.",
	Long:  `Run the goldilocks dashboard that will show recommendations.`,
	Run: func(cmd *cobra.Command, args []string) {
		var validBasePath = validateBasePath(basePath)

		// Open the history Store once and share it between the in-process
		// collector and the dashboard router, so both halves talk to the same
		// connection pool (avoids two opens against the same SQLite file).
		var historyStore *history.Store
		if dashboardHistoryDB != "" {
			hs, err := history.Open(dashboardHistoryDB)
			if err != nil {
				klog.Errorf("history: open %q failed: %v — trends disabled", dashboardHistoryDB, err)
			} else {
				historyStore = hs
				defer func() {
					if err := hs.Close(); err != nil {
						klog.Errorf("history: close store: %v", err)
					}
				}()
			}
		}

		// In-process collector — writes snapshots into the same Store the
		// router serves from. Recommended deployment: dashboard runs alone
		// with --history-db and --collect-history=true (default), avoiding
		// the need for a shared PVC between controller and dashboard.
		historyCtx, cancelHistory := context.WithCancel(context.Background())
		defer cancelHistory()
		if historyStore != nil && dashboardCollectHistory {
			collector := &history.Collector{
				Store:     historyStore,
				Interval:  dashboardHistoryInterval,
				Retention: dashboardHistoryRetention,
				NewSummary: func() (summary.Summary, error) {
					return summary.NewSummarizer().GetSummary()
				},
			}
			go collector.Run(historyCtx)
		}

		router := dashboard.GetRouter(
			dashboard.OnPort(serverPort),
			dashboard.BasePath(validBasePath),
			dashboard.ExcludeContainers(sets.New[string](strings.Split(excludeContainers, ",")...)),
			dashboard.OnByDefault(onByDefault),
			dashboard.ShowAllVPAs(showAllVPAs),
			dashboard.InsightsHost(insightsHost),
			dashboard.EnableCost(enableCost),
			dashboard.CacheTTL(cacheTTL),
			dashboard.EnableGzip(enableGzip),
			dashboard.HistoryDBPath(dashboardHistoryDB),
			dashboard.HistoryStore(historyStore),
		)
		http.Handle("/", router)
		klog.Infof("Starting goldilocks dashboard server on port %d and basePath %v", serverPort, validBasePath)

		// Stop the collector cleanly on SIGINT/SIGTERM. The HTTP server is
		// blocking ListenAndServe so we listen for signals in a goroutine.
		go func() {
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
			s := <-signals
			klog.Infof("Got signal: %v, stopping history collector", s)
			cancelHistory()
			os.Exit(0)
		}()

		klog.Fatalf("%v", http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil))
	},
}

func validateBasePath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	return path
}
