package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"ponctuel/services/api/graph"
	"ponctuel/services/ingester/store"
)

type HTTPConfig struct {
	Mode            string
	FreshnessWindow time.Duration
	MaxBodyBytes    int64
	QueryComplexity int
	AllowedOrigins  []string
}

type Metrics struct {
	requests      atomic.Uint64
	graphqlErrors atomic.Uint64
}

func NewMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if m == nil {
		m = NewMetrics()
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP ponctuel_api_http_requests_total HTTP requests received.\n# TYPE ponctuel_api_http_requests_total counter\nponctuel_api_http_requests_total %d\n", m.requests.Load())
	fmt.Fprintf(w, "# HELP ponctuel_api_graphql_errors_total GraphQL responses containing errors.\n# TYPE ponctuel_api_graphql_errors_total counter\nponctuel_api_graphql_errors_total %d\n", m.graphqlErrors.Load())
}

func NewHTTPHandler(cfg HTTPConfig, repository store.Repository, metrics *Metrics) http.Handler {
	if cfg.FreshnessWindow <= 0 {
		cfg.FreshnessWindow = 2 * time.Minute
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 1 << 20
	}
	if cfg.QueryComplexity <= 0 {
		cfg.QueryComplexity = 100
	}
	if metrics == nil {
		metrics = NewMetrics()
	}
	graphqlServer := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: graph.NewResolver(repository, cfg.Mode, cfg.FreshnessWindow),
	}))
	graphqlServer.AddTransport(transport.POST{})
	graphqlServer.Use(extension.FixedComplexityLimit(cfg.QueryComplexity))
	graphqlServer.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		response := next(ctx)
		if response != nil && len(response.Errors) > 0 {
			metrics.graphqlErrors.Add(1)
		}
		return response
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if strings.TrimSpace(cfg.Mode) == "" || repository == nil {
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
	mux.HandleFunc("/query", func(w http.ResponseWriter, request *http.Request) {
		metrics.requests.Add(1)
		if !allowedOrigin(w, request, cfg.AllowedOrigins) {
			return
		}
		if request.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", http.MethodPost+", "+http.MethodOptions)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if request.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
			return
		}
		if hasAPIKeyQueryParameter(request) {
			http.Error(w, "API credentials must not be sent to this endpoint\n", http.StatusBadRequest)
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, cfg.MaxBodyBytes)
		graphqlServer.ServeHTTP(w, request)
	})
	return countingHandler(mux, metrics)
}

func countingHandler(next http.Handler, metrics *Metrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" {
			metrics.requests.Add(1)
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOrigin(w http.ResponseWriter, r *http.Request, allowed []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if parsed, err := url.Parse(origin); err == nil && parsed.Host == r.Host {
		return true
	}
	for _, candidate := range allowed {
		if strings.TrimSpace(candidate) == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return true
		}
	}
	http.Error(w, "origin not allowed\n", http.StatusForbidden)
	return false
}

func hasAPIKeyQueryParameter(r *http.Request) bool {
	for key := range r.URL.Query() {
		if strings.EqualFold(key, "stm_api_key") || strings.EqualFold(key, "api_key") || strings.EqualFold(key, "apikey") {
			return true
		}
	}
	return false
}
