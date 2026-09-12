package store

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"ponctuel/services/ingester/domain"
)

func TestPostgresRepositoryContract(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	applyTestMigrations(t, ctx, pool)
	resetTestData(t, ctx, pool)

	repository := NewPostgresRepository(pool, 2*time.Minute)
	now := time.Now().UTC().Truncate(time.Microsecond)
	snapshot := domain.FeedSnapshot{
		FeedType:        domain.FeedTypeTripUpdates,
		RecordedAt:      now,
		SourceTimestamp: now,
		PayloadHash:     domain.HashPayload([]byte("snapshot-1")),
	}
	snapshotID, inserted, err := repository.InsertSnapshot(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted || snapshotID == 0 {
		t.Fatalf("first snapshot = id %d, inserted %t; want inserted snapshot", snapshotID, inserted)
	}
	duplicateID, duplicateInserted, err := repository.InsertSnapshot(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if duplicateInserted || duplicateID != 0 {
		t.Fatalf("duplicate snapshot = id %d, inserted %t; want zero, false", duplicateID, duplicateInserted)
	}

	events := []domain.Event{{
		Kind:         domain.EventKindPrediction,
		SnapshotHash: snapshot.PayloadHash,
		FeedType:     snapshot.FeedType,
		EntityID:     "trip-entity-1",
		RecordedAt:   now,
		VehicleID:    "bus-1",
		TripID:       "trip-1",
		RouteID:      "51",
		StopID:       "stop-1",
		StopSequence: 7,
		PredictedAt:  now,
		DelaySeconds: -90,
		HasDelay:     true,
	}}
	insertedEvents, err := repository.InsertEvents(ctx, snapshotID, events)
	if err != nil {
		t.Fatal(err)
	}
	if insertedEvents != 1 {
		t.Fatalf("inserted events = %d, want 1", insertedEvents)
	}
	arrival := domain.ArrivalObserved{
		TripID:       "trip-1",
		ServiceDate:  now.Format("2006-01-02"),
		StopID:       "stop-1",
		StopSequence: 7,
		ObservedAt:   now.Add(time.Minute),
		Method:       domain.ArrivalMethodLastUpdate,
		Confidence:   0.7,
		Reason:       "test arrival",
	}
	insertedArrival, err := repository.InsertArrival(ctx, arrival)
	if err != nil || !insertedArrival {
		t.Fatalf("first arrival inserted = %t, err %v; want true", insertedArrival, err)
	}
	duplicateArrival, err := repository.InsertArrival(ctx, arrival)
	if err != nil || duplicateArrival {
		t.Fatalf("duplicate arrival inserted = %t, err %v; want false", duplicateArrival, err)
	}
	summaries, err := repository.ErrorSummary(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].SampleCount != 1 || summaries[0].HorizonSeconds != 0 || summaries[0].MeanErrorSeconds != 60 || summaries[0].OnTimeRate != 1 {
		t.Fatalf("error summaries = %#v, want one one-minute on-time sample", summaries)
	}

	count, err := repository.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}

	freshSnapshot := domain.FeedSnapshot{
		FeedType:    domain.FeedTypeVehiclePositions,
		RecordedAt:  now,
		PayloadHash: domain.HashPayload([]byte("snapshot-fresh")),
	}
	freshID, freshInserted, err := repository.InsertSnapshot(ctx, freshSnapshot)
	if err != nil || !freshInserted {
		t.Fatalf("fresh snapshot = id %d, inserted %t, err %v", freshID, freshInserted, err)
	}
	staleRecordedAt := now.Add(-2 * time.Hour)
	staleSnapshot := domain.FeedSnapshot{
		FeedType:    domain.FeedTypeVehiclePositions,
		RecordedAt:  staleRecordedAt,
		PayloadHash: domain.HashPayload([]byte("snapshot-stale")),
	}
	staleID, staleInserted, err := repository.InsertSnapshot(ctx, staleSnapshot)
	if err != nil || !staleInserted {
		t.Fatalf("stale snapshot = id %d, inserted %t, err %v", staleID, staleInserted, err)
	}
	vehicleEvents := []domain.Event{
		{
			Kind:       domain.EventKindVehiclePosition,
			EntityID:   "vehicle-fresh",
			RecordedAt: now,
			VehicleID:  "bus-fresh",
			TripID:     "trip-1",
			RouteID:    "51",
			Latitude:   45.5017,
			Longitude:  -73.5673,
		},
		{
			Kind:       domain.EventKindVehiclePosition,
			EntityID:   "vehicle-stale",
			RecordedAt: staleRecordedAt,
			VehicleID:  "bus-stale",
			TripID:     "trip-2",
			RouteID:    "52",
			Latitude:   45.5,
			Longitude:  -73.5,
		},
	}
	if _, err := repository.InsertEvents(ctx, freshID, vehicleEvents[:1]); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.InsertEvents(ctx, staleID, vehicleEvents[1:]); err != nil {
		t.Fatal(err)
	}

	vehicles, err := repository.LatestVehicles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(vehicles) != 1 || vehicles[0].VehicleID != "bus-fresh" {
		t.Fatalf("latest vehicles = %+v, want only bus-fresh", vehicles)
	}
}

func resetTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		TRUNCATE prediction_error, arrival_observed, prediction, vehicle_position, feed_snapshot
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("reset test data: %v", err)
	}
}

func applyTestMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	migrationPaths, err := filepath.Glob(filepath.Join(workingDirectory, "..", "..", "..", "db", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(migrationPaths)
	if len(migrationPaths) != 6 {
		t.Fatalf("migration count = %d, want 6", len(migrationPaths))
	}
	for _, migrationPath := range migrationPaths {
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(migrationPath), err)
		}
	}
}
