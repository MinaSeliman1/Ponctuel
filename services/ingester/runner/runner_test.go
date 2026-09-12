package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"ponctuel/services/bus"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/domain"
	"ponctuel/services/ingester/fetch"
)

func TestRunCollectsImmediatelyAndStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	source := &fakeSource{snapshots: testSnapshots(), cancelAfter: 2, cancel: cancel}
	repository := &fakeRepository{}
	cfg := config.Config{PollInterval: time.Hour}

	if err := Run(ctx, cfg, source, repository); err != nil {
		t.Fatal(err)
	}
	if source.fetches != 2 {
		t.Fatalf("fetches = %d, want immediate collection of both feeds", source.fetches)
	}
	if repository.insertedEvents != 2 {
		t.Fatalf("inserted events = %d, want 2", repository.insertedEvents)
	}
}

func TestCollectOnceSkipsEventsForDuplicateSnapshot(t *testing.T) {
	runner := New(config.Config{}, &fakeSource{snapshots: testSnapshots()}, &fakeRepository{})
	if err := runner.collectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runner.collectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	repository := runner.repository.(*fakeRepository)
	if repository.snapshotCalls != 4 {
		t.Fatalf("snapshot calls = %d, want 4", repository.snapshotCalls)
	}
	if repository.insertedEvents != 2 {
		t.Fatalf("inserted events = %d, want duplicate cycle skipped", repository.insertedEvents)
	}
}

func TestCollectOncePublishesAfterPersistingEvents(t *testing.T) {
	publisher := &fakePublisher{}
	runner := NewWithMetricsAndPublisher(config.Config{}, &fakeSource{snapshots: testSnapshots()}, &fakeRepository{}, nil, publisher)

	if err := runner.collectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if publisher.published != 2 {
		t.Fatalf("published events = %d, want 2", publisher.published)
	}
	if publisher.topics["trip-updates"] != 1 || publisher.topics["vehicle-positions"] != 1 {
		t.Fatalf("published topics = %#v, want one event per topic", publisher.topics)
	}
}

func TestRunTickerHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	runner := New(config.Config{PollInterval: 5 * time.Millisecond}, &fakeSource{snapshots: testSnapshots()}, &fakeRepository{})

	if err := runner.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if runner.metrics.Fetches() < 2 {
		t.Fatalf("fetches = %d, want immediate collection", runner.metrics.Fetches())
	}
}

type fakeSource struct {
	mu          sync.Mutex
	snapshots   map[domain.FeedType]domain.FeedSnapshot
	fetches     int
	cancelAfter int
	cancel      context.CancelFunc
}

func (s *fakeSource) Fetch(_ context.Context, feedType domain.FeedType) (domain.FeedSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fetches++
	if s.cancelAfter > 0 && s.fetches >= s.cancelAfter && s.cancel != nil {
		s.cancel()
	}
	return s.snapshots[feedType], nil
}

type fakeRepository struct {
	mu             sync.Mutex
	seen           map[string]bool
	snapshotCalls  int
	insertedEvents int
}

type fakePublisher struct {
	published int
	topics    map[string]int
}

func (p *fakePublisher) Publish(_ context.Context, topic, _ string, _ domain.Event) error {
	if p.topics == nil {
		p.topics = make(map[string]int)
	}
	p.published++
	p.topics[topic]++
	return nil
}

func (p *fakePublisher) Close() error { return nil }

func (r *fakeRepository) InsertSnapshot(_ context.Context, snapshot domain.FeedSnapshot) (int64, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen == nil {
		r.seen = make(map[string]bool)
	}
	r.snapshotCalls++
	key := string(snapshot.FeedType) + ":" + snapshot.PayloadHash
	if r.seen[key] {
		return 0, false, nil
	}
	r.seen[key] = true
	return int64(r.snapshotCalls), true, nil
}

func (r *fakeRepository) InsertEvents(_ context.Context, _ int64, events []domain.Event) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.insertedEvents += len(events)
	return len(events), nil
}

func (r *fakeRepository) CountEvents(context.Context) (int64, error) { return 0, nil }
func (r *fakeRepository) LatestVehicles(context.Context) ([]domain.Vehicle, error) {
	return nil, nil
}
func (r *fakeRepository) Ping(context.Context) error { return nil }

func testSnapshots() map[domain.FeedType]domain.FeedSnapshot {
	now := time.Now().UTC()
	return map[domain.FeedType]domain.FeedSnapshot{
		domain.FeedTypeTripUpdates: {
			FeedType:    domain.FeedTypeTripUpdates,
			RecordedAt:  now,
			PayloadHash: domain.HashPayload([]byte("trip-update")),
			Entities: []domain.Entity{{
				ID: "trip-entity",
				TripUpdate: &domain.TripUpdate{TripID: "trip-1", StopTimeUpdates: []domain.StopTimeUpdate{{
					StopID: "stop-1", PredictedAt: now, HasDelay: true, DelaySeconds: -90,
				}}},
			}},
		},
		domain.FeedTypeVehiclePositions: {
			FeedType:    domain.FeedTypeVehiclePositions,
			RecordedAt:  now,
			PayloadHash: domain.HashPayload([]byte("vehicle-position")),
			Entities: []domain.Entity{{
				ID: "vehicle-entity",
				VehiclePosition: &domain.VehiclePosition{
					VehicleID: "bus-1", Latitude: 45.5, Longitude: -73.5, HasPosition: true, RecordedAt: now,
				},
			}},
		},
	}
}

var _ fetch.Source = (*fakeSource)(nil)
var _ bus.Publisher = (*fakePublisher)(nil)
