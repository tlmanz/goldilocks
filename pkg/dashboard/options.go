package dashboard

import (
	"time"

	"github.com/fairwindsops/goldilocks/pkg/history"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Option is a Functional options
type Option func(*Options)

// Options are options for getting and caching the Summarizer's VPAs
type Options struct {
	Port               int
	BasePath           string
	VpaLabels          map[string]string
	ExcludedContainers sets.Set[string]
	OnByDefault        bool
	ShowAllVPAs        bool
	InsightsHost       string
	EnableCost         bool
	// CacheTTL controls how long the dashboard summary is memoized in memory.
	// Zero disables the cache. Defaults to 30s.
	CacheTTL time.Duration
	// EnableGzip toggles gzip response compression on the dashboard router.
	EnableGzip bool
	// HistoryDBPath, when non-empty, points at a SQLite database used for
	// trend history. The dashboard process can both write to it (when the
	// in-process collector is enabled) and read from it via /api/history.
	HistoryDBPath string
	// HistoryStore, when non-nil, is an already-opened store the dashboard
	// router should serve from. Takes precedence over HistoryDBPath so that
	// the dashboard cmd can share a single Store with an in-process collector
	// (avoiding two opens of the same SQLite file in the same process).
	HistoryStore *history.Store
}

// default options for the dashboard
func defaultOptions() *Options {
	return &Options{
		Port:               8080,
		BasePath:           "/",
		VpaLabels:          utils.VPALabels,
		ExcludedContainers: sets.Set[string]{},
		OnByDefault:        false,
		ShowAllVPAs:        false,
		EnableCost:         true,
		CacheTTL:           30 * time.Second,
		EnableGzip:         true,
	}
}

// OnPort is an Option for running the dashboard on a different port
func OnPort(port int) Option {
	return func(opts *Options) {
		opts.Port = port
	}
}

// ExcludeContainers is an Option for excluding containers in the dashboard summary
func ExcludeContainers(excludedContainers sets.Set[string]) Option {
	return func(opts *Options) {
		opts.ExcludedContainers = excludedContainers
	}
}

// ForVPAsWithLabels Option for limiting the dashboard to certain VPAs matching the labels
func ForVPAsWithLabels(vpaLabels map[string]string) Option {
	return func(opts *Options) {
		opts.VpaLabels = vpaLabels
	}
}

// OnByDefault is an option for listing all namespaces in the dashboard unless explicitly excluded
func OnByDefault(onByDefault bool) Option {
	return func(opts *Options) {
		opts.OnByDefault = onByDefault
	}
}

func ShowAllVPAs(showAllVPAs bool) Option {
	return func(opts *Options) {
		opts.ShowAllVPAs = showAllVPAs
	}
}

func BasePath(basePath string) Option {
	return func(opts *Options) {
		opts.BasePath = basePath
	}
}

func InsightsHost(insightsHost string) Option {
	return func(opts *Options) {
		opts.InsightsHost = insightsHost
	}
}

func EnableCost(enableCost bool) Option {
	return func(opts *Options) {
		opts.EnableCost = enableCost
	}
}

// CacheTTL sets the dashboard summary cache duration. Zero disables caching.
func CacheTTL(ttl time.Duration) Option {
	return func(opts *Options) {
		opts.CacheTTL = ttl
	}
}

// EnableGzip toggles gzip response compression on the dashboard router.
func EnableGzip(enable bool) Option {
	return func(opts *Options) {
		opts.EnableGzip = enable
	}
}

// HistoryDBPath sets the SQLite history database path. Empty disables history.
func HistoryDBPath(path string) Option {
	return func(opts *Options) {
		opts.HistoryDBPath = path
	}
}

// HistoryStore lets the caller hand in an already-opened *history.Store so
// the router doesn't open a second connection pool against the same SQLite
// file (typically because the same process is also running the collector).
func HistoryStore(store *history.Store) Option {
	return func(opts *Options) {
		opts.HistoryStore = store
	}
}
