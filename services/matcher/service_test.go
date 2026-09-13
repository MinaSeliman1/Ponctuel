package matcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"ponctuel/services/bus"
	"ponctuel/services/events"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/domain"
)

func TestMatcherHTTPHandlerReadinessAndMetrics(t *testing.T) {
	repository := &serviceRepository{pingErr: context.Canceled}
	metrics := NewMetrics()
	ready := false
	handler := NewHTTPHandler(repository, metrics, func() bool { return ready })

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("not-ready status = %d, want 503", response.Code)
	}
	ready = true
	repository.pingErr = nil
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want 200", response.Code)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK || !containsAll(response.Body.String(), "ponctuel_matcher_events_total", "ponctuel_matcher_arrivals_total", "ponctuel_matcher_duplicates_total", "ponctuel_matcher_errors_total") {
		t.Fatalf("metrics response = %q", response.Body.String())
	}
}

func TestServiceConsumesMessageAndCommitsArrival(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	payload, err := encodeTestEvent(domain.Event{Kind: domain.EventKindPrediction, RecordedAt: now, TripID: "trip-1", StopID: "stop-1", PredictedAt: now.Add(-time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	reader := &fakeReader{messages: []kafka.Message{{Value: payload}}}
	repository := &serviceRepository{}
	service := NewService(config.Config{GeofenceRadiusMeters: 60}, repository, []bus.Reader{reader}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for len(repository.arrivals) == 0 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	if err := service.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(repository.arrivals) != 1 || reader.commits != 1 {
		t.Fatalf("arrivals = %#v, commits = %d", repository.arrivals, reader.commits)
	}
}

func encodeTestEvent(event domain.Event) ([]byte, error) {
	return events.Encode(event)
}

func containsAll(value string, wanted ...string) bool {
	for _, item := range wanted {
		if !strings.Contains(value, item) {
			return false
		}
	}
	return true
}

type fakeReader struct {
	messages []kafka.Message
	index    int
	commits  int
}

func (r *fakeReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if r.index < len(r.messages) {
		message := r.messages[r.index]
		r.index++
		return message, nil
	}
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (r *fakeReader) CommitMessages(context.Context, ...kafka.Message) error {
	r.commits++
	return nil
}

func (r *fakeReader) Close() error { return nil }

type serviceRepository struct {
	pingErr  error
	arrivals []domain.ArrivalObserved
}

func (r *serviceRepository) Ping(context.Context) error { return r.pingErr }
func (r *serviceRepository) InsertSnapshot(context.Context, domain.FeedSnapshot) (int64, bool, error) {
	return 0, false, nil
}
func (r *serviceRepository) InsertEvents(context.Context, int64, []domain.Event) (int, error) {
	return 0, nil
}
func (r *serviceRepository) CountEvents(context.Context) (int64, error) { return 0, nil }
func (r *serviceRepository) LatestSnapshotAt(context.Context) (time.Time, error) {
	return time.Time{}, nil
}
func (r *serviceRepository) LatestVehicles(context.Context) ([]domain.Vehicle, error) {
	return nil, nil
}
func (r *serviceRepository) InsertArrival(_ context.Context, arrival domain.ArrivalObserved) (bool, error) {
	r.arrivals = append(r.arrivals, arrival)
	return true, nil
}
func (r *serviceRepository) LatestStops(context.Context) ([]domain.Stop, error) {
	return nil, nil
}
func (r *serviceRepository) ErrorSummary(context.Context, int) ([]domain.ErrorSummary, error) {
	return nil, nil
}
