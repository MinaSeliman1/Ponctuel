package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"os/signal"

	"ponctuel/services/api"
	"ponctuel/services/bus"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/fetch"
	"ponctuel/services/ingester/runner"
	"ponctuel/services/ingester/store"
	"ponctuel/services/matcher"
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
	if cfg.AppEnv == "stm" {
		imported, err := repository.EnsureStaticSchedule(ctx, cfg.MatcherStopsURL)
		if err != nil {
			log.Fatal(err)
		}
		if imported {
			log.Printf("imported STM static schedule for realtime delay calculation")
		}
	}

	messageBus := bus.NewMemoryBus(1024)
	defer messageBus.Close()
	source := fetch.New(fetch.Config{
		Mode:                cfg.AppEnv,
		FixtureDir:          cfg.FixtureDir,
		TripUpdatesURL:      cfg.STMTripUpdatesURL,
		VehiclePositionsURL: cfg.STMVehicleURL,
		APIKey:              cfg.STMAPIKey,
		APIKeyHeader:        cfg.STMAPIKeyHeader,
	})
	ingesterRunner := runner.NewWithMetricsAndPublisher(cfg, source, repository, runner.NewMetrics(), messageBus)
	matcherService := matcher.NewService(cfg, repository, messageBus.Readers(), matcher.NewMetrics())

	apiMetrics := api.NewMetrics()
	apiServer := &http.Server{
		Addr:              apiHTTPAddr(os.LookupEnv),
		Handler:           api.NewHTTPHandler(api.HTTPConfig{Mode: cfg.AppEnv, FreshnessWindow: cfg.FreshnessWindow, AllowedOrigins: splitOrigins(os.Getenv("API_ALLOWED_ORIGINS"))}, repository, apiMetrics),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 3)
	go func() {
		if err := apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() { errCh <- ingesterRunner.Run(ctx) }()
	go func() { errCh <- matcherService.Run(ctx) }()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			log.Printf("public service stopped: %v", err)
		}
		stop()
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(shutdownContext); err != nil {
		log.Printf("API shutdown: %v", err)
	}
	_ = matcherService.Close()
}

func apiHTTPAddr(lookup func(string) (string, bool)) string {
	if lookup != nil {
		if value, ok := lookup("API_HTTP_ADDR"); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ":8080"
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
