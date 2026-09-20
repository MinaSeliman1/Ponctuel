package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ponctuel/services/api"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repository, err := store.ConnectWithSourceMode(ctx, cfg.DatabaseURL, cfg.FreshnessWindow, cfg.AppEnv)
	if err != nil {
		log.Fatal(err)
	}
	defer repository.Close()

	metrics := api.NewMetrics()
	server := &http.Server{
		Addr:              valueOr(os.Getenv("API_HTTP_ADDR"), ":8080"),
		Handler:           api.NewHTTPHandler(api.HTTPConfig{Mode: cfg.AppEnv, FreshnessWindow: cfg.FreshnessWindow, AllowedOrigins: splitOrigins(os.Getenv("API_ALLOWED_ORIGINS"))}, repository, metrics),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			log.Printf("API stopped: %v", err)
		}
		stop()
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("API shutdown: %v", err)
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func splitOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
