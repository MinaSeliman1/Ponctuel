package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/gtfs"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <gtfs.zip> <feed-version>", os.Args[0])
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	database, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	archive, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer archive.Close()
	if err := gtfs.Import(ctx, database, archive, os.Args[2]); err != nil {
		log.Fatal(err)
	}
	log.Printf("imported GTFS feed version %s", os.Args[2])
}
