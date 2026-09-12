package matcher

import (
	"context"
	"net/http"
	"time"

	"ponctuel/services/ingester/store"
)

func NewHTTPHandler(repository store.Repository, metrics *Metrics, ready func() bool) http.Handler {
	if metrics == nil {
		metrics = NewMetrics()
	}
	if ready == nil {
		ready = func() bool { return false }
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready() || repository == nil {
			http.Error(w, "not ready\n", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := repository.Ping(ctx); err != nil {
			http.Error(w, "not ready\n", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.Handle("/metrics", metrics)
	return mux
}
