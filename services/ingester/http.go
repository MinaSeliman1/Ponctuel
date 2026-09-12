package ingester

import (
	"context"
	"net/http"
	"time"

	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/runner"
	"ponctuel/services/ingester/store"
)

func NewHTTPHandler(cfg config.Config, repository store.Repository, metrics *runner.Metrics) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(response http.ResponseWriter, _ *http.Request) {
		if err := cfg.Validate(); err != nil || repository == nil {
			http.Error(response, "not ready\n", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := repository.Ping(ctx); err != nil {
			http.Error(response, "not ready\n", http.StatusServiceUnavailable)
			return
		}
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ready\n"))
	})
	if metrics == nil {
		metrics = runner.NewMetrics()
	}
	mux.Handle("/metrics", metrics)
	return mux
}
