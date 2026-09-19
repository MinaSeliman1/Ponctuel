package bus

import (
	"context"
	"testing"
	"time"

	"ponctuel/services/events"
	"ponctuel/services/ingester/domain"
)

func TestMemoryBusPublishesAnEventToTheMatchingReader(t *testing.T) {
	messageBus := NewMemoryBus(1)
	readers := messageBus.Readers()
	if len(readers) != 2 {
		t.Fatalf("Readers() returned %d readers, want 2", len(readers))
	}

	event := domain.Event{
		Kind:         domain.EventKindPrediction,
		SnapshotHash: "fixture-snapshot",
		EntityID:     "entity-1",
		RecordedAt:   time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC),
		PredictedAt:  time.Date(2026, 9, 19, 11, 59, 0, 0, time.UTC),
		TripID:       "trip-1",
		StopID:       "stop-1",
	}
	if err := messageBus.Publish(context.Background(), events.TopicTripUpdates, "trip-1", event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	message, err := readers[0].FetchMessage(context.Background())
	if err != nil {
		t.Fatalf("FetchMessage() error = %v", err)
	}
	decoded, err := events.Decode(message.Value)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded.TripID != event.TripID || decoded.StopID != event.StopID {
		t.Fatalf("decoded event = %#v, want %#v", decoded, event)
	}
}

func TestMemoryBusReaderStopsWhenTheBusCloses(t *testing.T) {
	messageBus := NewMemoryBus(1)
	readers := messageBus.Readers()
	if err := messageBus.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := readers[0].FetchMessage(context.Background()); err == nil {
		t.Fatal("FetchMessage() succeeded after the bus closed")
	}
}
