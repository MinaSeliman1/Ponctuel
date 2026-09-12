package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestNormalizeTripUpdatePreservesNegativeDelay(t *testing.T) {
	snapshot := FeedSnapshot{
		FeedType:   FeedTypeTripUpdates,
		RecordedAt: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC),
		Entities: []Entity{{
			ID: "trip-entity-1",
			TripUpdate: &TripUpdate{
				TripID:    "trip-1",
				RouteID:   "51",
				VehicleID: "bus-1",
				StopTimeUpdates: []StopTimeUpdate{{
					StopID:       "stop-1",
					StopSequence: 7,
					PredictedAt:  time.Date(2026, 9, 11, 12, 2, 0, 0, time.UTC),
					DelaySeconds: -90,
					HasDelay:     true,
				}},
			},
		}},
	}

	events, err := Normalize(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if got := events[0].DelaySeconds; got != -90 {
		t.Fatalf("delay = %d, want -90", got)
	}
	if got := events[0].Kind; got != EventKindPrediction {
		t.Fatalf("kind = %q, want %q", got, EventKindPrediction)
	}
}

func TestNormalizeVehiclePositionPreservesCoordinates(t *testing.T) {
	snapshot := FeedSnapshot{
		FeedType:   FeedTypeVehiclePositions,
		RecordedAt: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC),
		Entities: []Entity{{
			ID: "vehicle-entity-1",
			VehiclePosition: &VehiclePosition{
				VehicleID:  "bus-1",
				TripID:     "trip-1",
				RouteID:    "51",
				Latitude:   45.5017,
				Longitude:  -73.5673,
				RecordedAt: time.Date(2026, 9, 11, 11, 59, 59, 0, time.UTC),
			},
		}},
	}

	events, err := Normalize(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if got := events[0].Latitude; got != 45.5017 {
		t.Fatalf("latitude = %f, want 45.5017", got)
	}
	if got := events[0].Longitude; got != -73.5673 {
		t.Fatalf("longitude = %f, want -73.5673", got)
	}
}

func TestNormalizeSkipsEntityWithoutRelevantMessage(t *testing.T) {
	snapshot := FeedSnapshot{
		FeedType:   FeedTypeVehiclePositions,
		RecordedAt: time.Now().UTC(),
		Entities:   []Entity{{ID: "empty-entity"}},
	}

	events, err := Normalize(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestNormalizeRejectsInvalidCoordinates(t *testing.T) {
	snapshot := FeedSnapshot{
		FeedType:   FeedTypeVehiclePositions,
		RecordedAt: time.Now().UTC(),
		Entities: []Entity{{
			ID: "vehicle-entity-1",
			VehiclePosition: &VehiclePosition{
				VehicleID: "bus-1",
				Latitude:  95,
				Longitude: 0,
			},
		}},
	}

	if _, err := Normalize(snapshot); err == nil {
		t.Fatal("Normalize returned nil error for invalid coordinates")
	}
}

func TestHashPayloadIsDeterministic(t *testing.T) {
	payload := []byte("ponctuel-fixture")
	wantBytes := sha256.Sum256(payload)
	want := hex.EncodeToString(wantBytes[:])
	if got := HashPayload(payload); got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
	if HashPayload(payload) != HashPayload(payload) {
		t.Fatal("hash changed between identical calls")
	}
}
