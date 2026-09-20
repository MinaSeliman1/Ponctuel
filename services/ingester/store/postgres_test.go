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
	}, {
		Kind:         domain.EventKindPrediction,
		SnapshotHash: snapshot.PayloadHash,
		FeedType:     snapshot.FeedType,
		EntityID:     "trip-entity-2",
		RecordedAt:   now.Add(-5 * time.Minute),
		VehicleID:    "bus-1",
		TripID:       "trip-1",
		RouteID:      "51",
		StopID:       "stop-1",
		StopSequence: 7,
		PredictedAt:  now.Add(2 * time.Minute),
		DelaySeconds: -30,
		HasDelay:     true,
	}}
	insertedEvents, err := repository.InsertEvents(ctx, snapshotID, events)
	if err != nil {
		t.Fatal(err)
	}
	if insertedEvents != 2 {
		t.Fatalf("inserted events = %d, want 2", insertedEvents)
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
	if len(summaries) != 2 || summaries[0].SampleCount != 1 || summaries[0].HorizonSeconds != 0 || summaries[0].MeanErrorSeconds != 60 || summaries[0].OnTimeRate != 1 || summaries[1].SampleCount != 1 || summaries[1].HorizonSeconds != 420 || summaries[1].MeanErrorSeconds != -60 || summaries[1].OnTimeRate != 1 {
		t.Fatalf("error summaries = %#v, want two one-sample on-time horizons", summaries)
	}

	count, err := repository.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("event count = %d, want 2", count)
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
			VehicleID:  "bus-1",
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
	if len(vehicles) != 1 || vehicles[0].VehicleID != "bus-1" {
		t.Fatalf("latest vehicles = %+v, want only bus-1", vehicles)
	}
	if !vehicles[0].HasDelay || vehicles[0].DelaySeconds != -90 {
		t.Fatalf("latest vehicle delay = %+v, want -90 seconds from matching trip update", vehicles[0])
	}
}

func TestPostgresRepositoryScopesDataToConfiguredSourceMode(t *testing.T) {
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

	now := time.Now().UTC().Truncate(time.Microsecond)
	fixtureRepository := NewPostgresRepositoryWithSourceMode(pool, 2*time.Minute, "fixture")
	stmRepository := NewPostgresRepositoryWithSourceMode(pool, 2*time.Minute, "stm")
	snapshotIDs := make(map[string]int64, 2)
	for _, snapshot := range []domain.FeedSnapshot{
		{FeedType: domain.FeedTypeVehiclePositions, RecordedAt: now, PayloadHash: domain.HashPayload([]byte("fixture-source")), SourceMode: "fixture"},
		{FeedType: domain.FeedTypeVehiclePositions, RecordedAt: now.Add(time.Second), PayloadHash: domain.HashPayload([]byte("stm-source")), SourceMode: "stm"},
	} {
		repository := fixtureRepository
		if snapshot.SourceMode == "stm" {
			repository = stmRepository
		}
		snapshotID, inserted, insertErr := repository.InsertSnapshot(ctx, snapshot)
		if insertErr != nil || !inserted {
			t.Fatalf("insert source snapshot %q = inserted %t, err %v", snapshot.SourceMode, inserted, insertErr)
		}
		snapshotIDs[snapshot.SourceMode] = snapshotID
	}
	for mode, repository := range map[string]*PostgresRepository{"fixture": fixtureRepository, "stm": stmRepository} {
		if _, err := repository.InsertEvents(ctx, snapshotIDs[mode], []domain.Event{{
			Kind:       domain.EventKindVehiclePosition,
			EntityID:   mode + "-vehicle",
			RecordedAt: now,
			VehicleID:  mode + "-vehicle",
			Latitude:   45.5017,
			Longitude:  -73.5673,
		}}); err != nil {
			t.Fatalf("insert %s source event: %v", mode, err)
		}
	}

	fixtureCount, err := fixtureRepository.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stmCount, err := stmRepository.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fixtureCount != 1 || stmCount != 1 {
		t.Fatalf("source counts = fixture %d, stm %d; want one event per mode", fixtureCount, stmCount)
	}

	fixtureLatest, err := fixtureRepository.LatestSnapshotAt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stmLatest, err := stmRepository.LatestSnapshotAt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fixtureLatest.IsZero() || stmLatest.IsZero() || !stmLatest.After(fixtureLatest) {
		t.Fatalf("source latest timestamps = fixture %v, stm %v; want separate non-zero values", fixtureLatest, stmLatest)
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
	if len(migrationPaths) != 7 {
		t.Fatalf("migration count = %d, want 7", len(migrationPaths))
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
