package matcher

import (
	"strings"
	"testing"
	"time"

	"ponctuel/services/ingester/domain"
)

func TestEngineEmitsLastUpdateForExpiredPrediction(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	engine := NewEngine(60)
	arrivals, err := engine.HandleEvent(domain.Event{
		Kind:         domain.EventKindPrediction,
		RecordedAt:   now,
		TripID:       "trip-1",
		StopID:       "stop-1",
		StopSequence: 2,
		PredictedAt:  now.Add(-time.Minute),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(arrivals) != 1 || arrivals[0].Method != domain.ArrivalMethodLastUpdate || arrivals[0].Confidence != 0.70 {
		t.Fatalf("arrivals = %#v, want one last_update arrival", arrivals)
	}
}

func TestEngineEmitsGeofenceWithDistanceBasedConfidence(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	engine := NewEngine(60)
	_, err := engine.HandleEvent(domain.Event{
		Kind:         domain.EventKindPrediction,
		RecordedAt:   now,
		TripID:       "trip-1",
		StopID:       "stop-1",
		StopSequence: 1,
		PredictedAt:  now.Add(5 * time.Minute),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	arrivals, err := engine.HandleEvent(domain.Event{
		Kind:       domain.EventKindVehiclePosition,
		RecordedAt: now.Add(time.Minute),
		TripID:     "trip-1",
		VehicleID:  "bus-1",
		Latitude:   45.5017,
		Longitude:  -73.5673,
	}, []domain.Stop{{StopID: "stop-1", Latitude: 45.5017, Longitude: -73.5673}})
	if err != nil {
		t.Fatal(err)
	}
	if len(arrivals) != 1 || arrivals[0].Method != domain.ArrivalMethodGeofence || arrivals[0].Confidence != 1 {
		t.Fatalf("arrivals = %#v, want one high-confidence geofence arrival", arrivals)
	}
	if !strings.Contains(arrivals[0].Reason, "geofence") {
		t.Fatalf("reason = %q, want geofence explanation", arrivals[0].Reason)
	}
	duplicates, err := engine.HandleEvent(domain.Event{
		Kind:       domain.EventKindVehiclePosition,
		RecordedAt: now.Add(2 * time.Minute),
		TripID:     "trip-1",
		VehicleID:  "bus-1",
		Latitude:   45.5017,
		Longitude:  -73.5673,
	}, []domain.Stop{{StopID: "stop-1", Latitude: 45.5017, Longitude: -73.5673}})
	if err != nil {
		t.Fatal(err)
	}
	if len(duplicates) != 0 {
		t.Fatalf("repeated position arrivals = %#v, want none", duplicates)
	}
}

func TestEngineIgnoresOutsideGeofenceAndDifferentTrip(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	engine := NewEngine(60)
	_, _ = engine.HandleEvent(domain.Event{Kind: domain.EventKindPrediction, RecordedAt: now, TripID: "trip-1", StopID: "stop-1", PredictedAt: now.Add(time.Minute)}, nil)
	arrivals, err := engine.HandleEvent(domain.Event{Kind: domain.EventKindVehiclePosition, RecordedAt: now, TripID: "trip-2", VehicleID: "bus-2", Latitude: 45.5017, Longitude: -73.5673}, []domain.Stop{{StopID: "stop-1", Latitude: 45.5017, Longitude: -73.5673}})
	if err != nil {
		t.Fatal(err)
	}
	if len(arrivals) != 0 {
		t.Fatalf("different trip arrivals = %#v", arrivals)
	}
	arrivals, err = engine.HandleEvent(domain.Event{Kind: domain.EventKindVehiclePosition, RecordedAt: now, TripID: "trip-1", VehicleID: "bus-1", Latitude: 45.6, Longitude: -73.7}, []domain.Stop{{StopID: "stop-1", Latitude: 45.5017, Longitude: -73.5673}})
	if err != nil {
		t.Fatal(err)
	}
	if len(arrivals) != 0 {
		t.Fatalf("outside geofence arrivals = %#v", arrivals)
	}
}
