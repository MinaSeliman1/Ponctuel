package gtfs

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestValidateRequiredFilesRejectsIncompleteArchive(t *testing.T) {
	archive := buildGTFSArchive(t, map[string]string{"agency.txt": "agency_id,agency_name\na1,STM\n"})
	zipReader := openTestArchive(t, archive)
	if err := ValidateRequiredFiles(zipReader); err == nil || !strings.Contains(err.Error(), "routes.txt") {
		t.Fatalf("validation error = %v, want missing routes.txt", err)
	}
}

func TestImportLoadsRowsAndSupportsGTFSOvernightTimes(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	applyTestMigrations(t, ctx, database)
	if _, err := database.Exec(ctx, "TRUNCATE gtfs_feed_version CASCADE"); err != nil {
		t.Fatal(err)
	}

	if err := Import(ctx, database, bytes.NewReader(testArchive(t)), "2026-09-12"); err != nil {
		t.Fatal(err)
	}
	var agencies, routes, trips, stops, stopTimes, arrivalSeconds int
	err = database.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM gtfs_agency),
			(SELECT count(*) FROM gtfs_route),
			(SELECT count(*) FROM gtfs_trip),
			(SELECT count(*) FROM gtfs_stop),
			(SELECT count(*) FROM gtfs_stop_time),
			(SELECT arrival_seconds FROM gtfs_stop_time WHERE trip_id = 'trip-1' AND stop_sequence = 1)
	`).Scan(&agencies, &routes, &trips, &stops, &stopTimes, &arrivalSeconds)
	if err != nil {
		t.Fatal(err)
	}
	if agencies != 1 || routes != 1 || trips != 1 || stops != 2 || stopTimes != 2 {
		t.Fatalf("counts = agency %d route %d trip %d stop %d stop_time %d", agencies, routes, trips, stops, stopTimes)
	}
	if arrivalSeconds != 25*60*60+10*60 {
		t.Fatalf("arrival seconds = %d, want 90600", arrivalSeconds)
	}
	if err := Import(ctx, database, bytes.NewReader(testArchive(t)), "2026-09-12"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate import error = %v, want immutable feed version error", err)
	}
}

func TestImportReportsMalformedCSVFileAndRow(t *testing.T) {
	malformed := buildGTFSArchive(t, map[string]string{
		"agency.txt":     "agency_id,agency_name\na1,STM\n",
		"routes.txt":     "route_id,route_type\n51,not-a-number\n",
		"trips.txt":      "trip_id\ntrip-1\n",
		"stops.txt":      "stop_id,stop_name\ns1,Stop 1\ns2,Stop 2\n",
		"stop_times.txt": "trip_id,stop_id,stop_sequence,arrival_time,departure_time\ntrip-1,s1,1,25:10:00,25:11:00\n",
	})
	if err := Import(context.Background(), nil, bytes.NewReader(malformed), "2026-09-12"); err == nil || !strings.Contains(err.Error(), "routes.txt row 2 column route_type") {
		t.Fatalf("malformed CSV error = %v, want file and row context", err)
	}
}

func testArchive(t *testing.T) []byte {
	t.Helper()
	return buildGTFSArchive(t, map[string]string{
		"agency.txt":     "agency_id,agency_name,agency_url,agency_timezone\na1,STM,https://www.stm.info,America/Toronto\n",
		"routes.txt":     "route_id,agency_id,route_short_name,route_long_name,route_type\n51,a1,51,Edouard-Montpetit,3\n",
		"trips.txt":      "route_id,service_id,trip_id,trip_headsign,direction_id\n51,weekday,trip-1,Direction nord,0\n",
		"stops.txt":      "stop_id,stop_name,stop_lat,stop_lon\ns1,Stop 1,45.5,-73.5\ns2,Stop 2,45.51,-73.51\n",
		"stop_times.txt": "trip_id,arrival_time,departure_time,stop_id,stop_sequence\ntrip-1,25:10:00,25:11:00,s1,1\ntrip-1,25:20:00,25:21:00,s2,2\n",
	})
}

func buildGTFSArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func openTestArchive(t *testing.T, payload []byte) *zip.Reader {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func applyTestMigrations(t *testing.T, ctx context.Context, database *pgxpool.Pool) {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(workingDirectory, "..", "..", "..", "db", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	for _, path := range paths {
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(ctx, string(payload)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), err)
		}
	}
}
