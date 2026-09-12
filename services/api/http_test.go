package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ponctuel/services/ingester/domain"
)

func TestGraphQLDashboardVehiclesAndStaleFlag(t *testing.T) {
	now := time.Now().UTC().Add(-10 * time.Second)
	repository := &apiFakeRepository{
		eventCount:      7,
		latestCollected: now,
		vehicles: []domain.Vehicle{
			{VehicleID: "bus-1", RouteID: "51", TripID: "trip-1", Latitude: 45.5, Longitude: -73.5, RecordedAt: now, DelaySeconds: -30, HasDelay: true},
			{VehicleID: "bus-2", RouteID: "80", TripID: "trip-2", Latitude: 45.6, Longitude: -73.6, RecordedAt: now, DelaySeconds: 90, HasDelay: true},
		},
	}
	handler := NewHTTPHandler(HTTPConfig{Mode: "fixture", FreshnessWindow: time.Minute}, repository, nil)
	payload := `{"query":"{ dashboard { eventCount lastCollectedAt mode stale } vehicles(limit: 1) { vehicleId routeId tripId latitude longitude recordedAt delaySeconds } }"}`
	response := postGraphQL(t, handler, payload, "")
	if response.Code != http.StatusOK {
		t.Fatalf("GraphQL status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{`"eventCount":7`, `"mode":"fixture"`, `"stale":false`, `"vehicleId":"bus-1"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("GraphQL body = %s, missing %s", body, expected)
		}
	}
	if strings.Contains(body, "bus-2") {
		t.Fatalf("vehicle limit was ignored: %s", body)
	}
}

func TestGraphQLDashboardMarksOldDataStale(t *testing.T) {
	repository := &apiFakeRepository{eventCount: 1, latestCollected: time.Now().UTC().Add(-10 * time.Minute)}
	handler := NewHTTPHandler(HTTPConfig{Mode: "fixture", FreshnessWindow: time.Minute}, repository, nil)
	response := postGraphQL(t, handler, `{"query":"{ dashboard { stale } }"}`, "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"stale":true`) {
		t.Fatalf("stale response = %d %s", response.Code, response.Body.String())
	}
}

func TestGraphQLErrorSummaryMapsAndLimitsRows(t *testing.T) {
	repository := &apiFakeRepository{errorSummaries: []domain.ErrorSummary{
		{RouteID: "51", HorizonSeconds: 300, SampleCount: 4, MeanErrorSeconds: 42.5, OnTimeRate: 0.75},
		{RouteID: "80", HorizonSeconds: 600, SampleCount: 2, MeanErrorSeconds: -15, OnTimeRate: 1},
	}}
	handler := NewHTTPHandler(HTTPConfig{Mode: "fixture", FreshnessWindow: time.Minute}, repository, nil)
	response := postGraphQL(t, handler, `{"query":"{ errorSummary(limit: 1) { routeId horizonSeconds sampleCount meanErrorSeconds onTimeRate } }"}`, "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"routeId":"51"`) || strings.Contains(response.Body.String(), `"routeId":"80"`) {
		t.Fatalf("error summary response = %d %s", response.Code, response.Body.String())
	}
}

func TestGraphQLErrorSummaryRejectsNegativeLimit(t *testing.T) {
	handler := NewHTTPHandler(HTTPConfig{Mode: "fixture", FreshnessWindow: time.Minute}, &apiFakeRepository{}, nil)
	response := postGraphQL(t, handler, `{"query":"{ errorSummary(limit: -1) { horizonSeconds } }"}`, "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "error summary limit must not be negative") {
		t.Fatalf("negative error summary response = %d %s", response.Code, response.Body.String())
	}
}

func TestGraphQLReadyFailsWhenDatabasePingFails(t *testing.T) {
	repository := &apiFakeRepository{pingErr: context.Canceled}
	handler := NewHTTPHandler(HTTPConfig{Mode: "fixture", FreshnessWindow: time.Minute}, repository, nil)
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, want 503", response.Code)
	}
}

func TestGraphQLRejectsAPIKeyQueryParameter(t *testing.T) {
	handler := NewHTTPHandler(HTTPConfig{Mode: "fixture", FreshnessWindow: time.Minute}, &apiFakeRepository{}, nil)
	response := postGraphQL(t, handler, `{"query":"{ dashboard { mode } }"}`, "super-secret")
	if response.Code != http.StatusBadRequest || strings.Contains(response.Body.String(), "super-secret") {
		t.Fatalf("API key query response = %d %s", response.Code, response.Body.String())
	}
}

func postGraphQL(t *testing.T, handler http.Handler, payload, apiKey string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(payload))
	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.URL.RawQuery = "STM_API_KEY=" + apiKey
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type apiFakeRepository struct {
	eventCount      int64
	latestCollected time.Time
	vehicles        []domain.Vehicle
	errorSummaries  []domain.ErrorSummary
	pingErr         error
}

func (r *apiFakeRepository) Ping(context.Context) error                 { return r.pingErr }
func (r *apiFakeRepository) CountEvents(context.Context) (int64, error) { return r.eventCount, nil }
func (r *apiFakeRepository) LatestSnapshotAt(context.Context) (time.Time, error) {
	return r.latestCollected, nil
}
func (r *apiFakeRepository) LatestVehicles(context.Context) ([]domain.Vehicle, error) {
	return r.vehicles, nil
}
func (r *apiFakeRepository) InsertArrival(context.Context, domain.ArrivalObserved) (bool, error) {
	return false, nil
}
func (r *apiFakeRepository) LatestStops(context.Context) ([]domain.Stop, error) { return nil, nil }
func (r *apiFakeRepository) ErrorSummary(context.Context, int) ([]domain.ErrorSummary, error) {
	return r.errorSummaries, nil
}
func (r *apiFakeRepository) InsertSnapshot(context.Context, domain.FeedSnapshot) (int64, bool, error) {
	return 0, false, nil
}
func (r *apiFakeRepository) InsertEvents(context.Context, int64, []domain.Event) (int, error) {
	return 0, nil
}
