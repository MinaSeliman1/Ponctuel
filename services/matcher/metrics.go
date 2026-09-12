package matcher

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	events     atomic.Uint64
	arrivals   atomic.Uint64
	duplicates atomic.Uint64
	errors     atomic.Uint64
}

func NewMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if m == nil {
		m = NewMetrics()
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP ponctuel_matcher_events_total Normalized events consumed.\n# TYPE ponctuel_matcher_events_total counter\nponctuel_matcher_events_total %d\n", m.events.Load())
	fmt.Fprintf(w, "# HELP ponctuel_matcher_arrivals_total Arrivals inserted.\n# TYPE ponctuel_matcher_arrivals_total counter\nponctuel_matcher_arrivals_total %d\n", m.arrivals.Load())
	fmt.Fprintf(w, "# HELP ponctuel_matcher_duplicates_total Arrival duplicates skipped.\n# TYPE ponctuel_matcher_duplicates_total counter\nponctuel_matcher_duplicates_total %d\n", m.duplicates.Load())
	fmt.Fprintf(w, "# HELP ponctuel_matcher_errors_total Matcher processing errors.\n# TYPE ponctuel_matcher_errors_total counter\nponctuel_matcher_errors_total %d\n", m.errors.Load())
}

func (m *Metrics) Events() uint64 {
	if m == nil {
		return 0
	}
	return m.events.Load()
}
