package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ponctuel/services/ingester"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/fetch"
	"ponctuel/services/ingester/runner"
	"ponctuel/services/ingester/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repository, err := store.Connect(ctx, cfg.DatabaseURL, cfg.FreshnessWindow)
	if err != nil {
		log.Fatal(err)
	}
	defer repository.Close()

	source := fetch.New(fetch.Config{
		Mode:                cfg.AppEnv,
		FixtureDir:          cfg.FixtureDir,
		TripUpdatesURL:      cfg.STMTripUpdatesURL,
		VehiclePositionsURL: cfg.STMVehicleURL,
		APIKey:              cfg.STMAPIKey,
		APIKeyHeader:        cfg.STMAPIKeyHeader,
	})
	metrics := runner.NewMetrics()
	ingesterRunner := runner.NewWithMetrics(cfg, source, repository, metrics)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: ingester.NewHTTPHandler(cfg, repository, metrics)}

	errCh := make(chan error, 2)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() { errCh <- ingesterRunner.Run(ctx) }()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			log.Printf("ingester stopped: %v", err)
		}
		stop()
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
}
