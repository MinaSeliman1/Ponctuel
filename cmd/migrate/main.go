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

const legacyBaselineVersion = "007_source_mode.sql"

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
	if err := ensureMigrationHistory(ctx, pool); err != nil {
		log.Fatalf("initialize migration history: %v", err)
	}
	if err := bootstrapLegacyMigrationHistory(ctx, pool, files); err != nil {
		log.Fatalf("bootstrap migration history: %v", err)
	}
	applied, err := loadAppliedMigrationVersions(ctx, pool)
	if err != nil {
		log.Fatalf("load migration history: %v", err)
	}

	for _, path := range files {
		version := filepath.Base(path)
		if _, ok := applied[version]; ok {
			log.Printf("skipped %s", version)
			continue
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("read migration %s: %v", version, err)
		}
		transaction, err := pool.Begin(ctx)
		if err != nil {
			log.Fatalf("begin migration %s: %v", version, err)
		}
		if _, err := transaction.Exec(ctx, string(contents)); err != nil {
			_ = transaction.Rollback(ctx)
			log.Fatalf("apply migration %s: %v", version, err)
		}
		if _, err := transaction.Exec(ctx, `
INSERT INTO public.schema_migration (version)
VALUES ($1)
ON CONFLICT (version) DO NOTHING`, version); err != nil {
			_ = transaction.Rollback(ctx)
			log.Fatalf("record migration %s: %v", version, err)
		}
		if err := transaction.Commit(ctx); err != nil {
			log.Fatalf("commit migration %s: %v", version, err)
		}
		applied[version] = struct{}{}
		log.Printf("applied %s", version)
	}
}

func ensureMigrationHistory(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS public.schema_migration (
    version text PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
)`)
	return err
}

func loadAppliedMigrationVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM public.schema_migration`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func bootstrapLegacyMigrationHistory(ctx context.Context, pool *pgxpool.Pool, files []string) error {
	var legacySchemaReady bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'feed_snapshot'
      AND column_name = 'source_mode'
)`).Scan(&legacySchemaReady); err != nil {
		return err
	}
	if !legacySchemaReady {
		return nil
	}

	for _, version := range legacyBaselineVersions(files) {
		if _, err := pool.Exec(ctx, `
INSERT INTO public.schema_migration (version)
VALUES ($1)
ON CONFLICT (version) DO NOTHING`, version); err != nil {
			return fmt.Errorf("record legacy migration %s: %w", version, err)
		}
	}
	return nil
}

func legacyBaselineVersions(files []string) []string {
	versions := make([]string, 0, len(files))
	for _, path := range files {
		version := filepath.Base(path)
		if version > legacyBaselineVersion {
			break
		}
		versions = append(versions, version)
	}
	return versions
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
