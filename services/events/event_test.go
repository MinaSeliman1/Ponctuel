package events

import (
	"testing"
	"time"

	"ponctuel/services/ingester/domain"
)

func TestEncodeDecodePreservesPrediction(t *testing.T) {
	recordedAt := time.Date(2026, 9, 12, 12, 0, 0, 123000000, time.FixedZone("EDT", -4*60*60))
	event := domain.Event{
		Kind:         domain.EventKindPrediction,
		SnapshotHash: "snapshot-hash",
		EntityID:     "entity-1",
		RecordedAt:   recordedAt,
		TripID:       "trip-1",
		StopID:       "stop-1",
		PredictedAt:  recordedAt.Add(2 * time.Minute),
		DelaySeconds: 45,
		HasDelay:     true,
	}
	payload, err := Encode(event)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := Decode(payload)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !decoded.RecordedAt.Equal(recordedAt.UTC()) || !decoded.PredictedAt.Equal(event.PredictedAt.UTC()) {
		t.Fatalf("decoded timestamps = %#v, want UTC timestamps", decoded)
	}
	if decoded.TripID != event.TripID || decoded.StopID != event.StopID || decoded.DelaySeconds != event.DelaySeconds {
		t.Fatalf("decoded event = %#v, want %#v", decoded, event)
	}
}

func TestTopicFor(t *testing.T) {
	if topic, _ := TopicFor(domain.EventKindPrediction); topic != TopicTripUpdates {
		t.Fatalf("prediction topic = %q", topic)
	}
	if topic, _ := TopicFor(domain.EventKindVehiclePosition); topic != TopicVehiclePositions {
		t.Fatalf("vehicle topic = %q", topic)
	}
}

func TestDecodeRejectsUnknownSchema(t *testing.T) {
	if _, err := Decode([]byte(`{"schema_version":99,"event":{}}`)); err == nil {
		t.Fatal("Decode() accepted an unknown schema")
	}
}
