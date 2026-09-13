package ingester

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/domain"
	"ponctuel/services/ingester/runner"
)

func TestHTTPHealthReadyAndMetricsEndpoints(t *testing.T) {
	repository := &healthRepository{}
	handler := NewHTTPHandler(testHTTPConfig(), repository, runner.NewMetrics())
	server := httptest.NewServer(handler)
	defer server.Close()

	assertStatus(t, server.URL+"/healthz", http.StatusOK)
	assertStatus(t, server.URL+"/readyz", http.StatusOK)
	response, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "ponctuel_ingester_fetch_total") {
		t.Fatalf("metrics response = %d %q", response.StatusCode, string(body))
	}
}

func TestHTTPReadyReturns503WhenDatabasePingFails(t *testing.T) {
	repository := &healthRepository{pingErr: context.Canceled}
	handler := NewHTTPHandler(testHTTPConfig(), repository, nil)
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, want 503", response.Code)
	}
}

func assertStatus(t *testing.T, endpoint string, want int) {
	t.Helper()
	response, err := http.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != want {
		t.Fatalf("GET %s status = %d, want %d", endpoint, response.StatusCode, want)
	}
}

func testHTTPConfig() config.Config {
	return config.Config{
		AppEnv:            "fixture",
		HTTPAddr:          ":8081",
		DatabaseURL:       "postgres://ponctuel:ponctuel@localhost:5432/ponctuel?sslmode=disable",
		PollInterval:      30 * time.Second,
		FreshnessWindow:   2 * time.Minute,
		FixtureDir:        "testdata/realtime",
		STMTripUpdatesURL: "https://api.stm.info/pub/od/gtfs-rt/ic/v2/tripUpdates",
		STMVehicleURL:     "https://api.stm.info/pub/od/gtfs-rt/ic/v2/vehiclePositions",
		STMAPIKeyHeader:   "apikey",
	}
}

type healthRepository struct {
	pingErr error
}

func (r *healthRepository) Ping(context.Context) error { return r.pingErr }
func (r *healthRepository) InsertSnapshot(context.Context, domain.FeedSnapshot) (int64, bool, error) {
	return 0, false, nil
}
func (r *healthRepository) InsertEvents(context.Context, int64, []domain.Event) (int, error) {
	return 0, nil
}
func (r *healthRepository) CountEvents(context.Context) (int64, error) { return 0, nil }
func (r *healthRepository) LatestVehicles(context.Context) ([]domain.Vehicle, error) {
	return nil, nil
}
func (r *healthRepository) InsertArrival(context.Context, domain.ArrivalObserved) (bool, error) {
	return false, nil
}
func (r *healthRepository) LatestStops(context.Context) ([]domain.Stop, error) { return nil, nil }
func (r *healthRepository) ErrorSummary(context.Context, int) ([]domain.ErrorSummary, error) {
	return nil, nil
}
