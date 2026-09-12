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

	"ponctuel/services/bus"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/store"
	"ponctuel/services/matcher"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if len(cfg.RedpandaBrokers) == 0 {
		log.Fatal("REDPANDA_BROKERS is required for matcher")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repository, err := store.Connect(ctx, cfg.DatabaseURL, cfg.FreshnessWindow)
	if err != nil {
		log.Fatal(err)
	}
	defer repository.Close()
	readers, err := bus.NewKafkaReaders(cfg.RedpandaBrokers, cfg.RedpandaGroupID)
	if err != nil {
		log.Fatal(err)
	}
	service := matcher.NewService(cfg, repository, readers, matcher.NewMetrics())
	defer service.Close()

	server := &http.Server{
		Addr:              cfg.MatcherHTTPAddr,
		Handler:           matcher.NewHTTPHandler(repository, service.Metrics(), service.Ready),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 2)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() { errCh <- service.Run(ctx) }()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			log.Printf("matcher stopped: %v", err)
		}
		stop()
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("matcher HTTP shutdown: %v", err)
	}
}
