package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	migrationsDirectory := valueOr(os.Getenv("MIGRATIONS_DIR"), "/app/db/migrations")
	files, err := orderedMigrationFiles(migrationsDirectory)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect PostgreSQL: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping PostgreSQL: %v", err)
	}

	for _, path := range files {
		contents, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("read migration %s: %v", filepath.Base(path), err)
		}
		transaction, err := pool.Begin(ctx)
		if err != nil {
			log.Fatalf("begin migration %s: %v", filepath.Base(path), err)
		}
		if _, err := transaction.Exec(ctx, string(contents)); err != nil {
			_ = transaction.Rollback(ctx)
			log.Fatalf("apply migration %s: %v", filepath.Base(path), err)
		}
		if err := transaction.Commit(ctx); err != nil {
			log.Fatalf("commit migration %s: %v", filepath.Base(path), err)
		}
		log.Printf("applied %s", filepath.Base(path))
	}
}

func orderedMigrationFiles(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".sql" {
			continue
		}
		files = append(files, filepath.Join(directory, entry.Name()))
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no SQL migrations found in %s", directory)
	}
	return files, nil
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
