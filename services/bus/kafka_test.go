package bus

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"ponctuel/services/events"
	"ponctuel/services/ingester/domain"
)

func TestMessageForEncodesVersionedEvent(t *testing.T) {
	event := domain.Event{
		Kind:       domain.EventKindVehiclePosition,
		RecordedAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC),
		VehicleID:  "bus-1",
		Latitude:   45.5,
		Longitude:  -73.5,
	}
	message, err := messageFor(event, "bus-1")
	if err != nil {
		t.Fatalf("messageFor() error = %v", err)
	}
	if string(message.Key) != "bus-1" {
		t.Fatalf("message key = %q", message.Key)
	}
	decoded, err := events.Decode(message.Value)
	if err != nil {
		t.Fatalf("events.Decode() error = %v", err)
	}
	if decoded.VehicleID != event.VehicleID {
		t.Fatalf("decoded vehicle = %#v, want %#v", decoded, event)
	}
}

func TestKafkaPublisherRequiresBroker(t *testing.T) {
	if _, err := NewKafkaPublisher(nil); err == nil {
		t.Fatal("NewKafkaPublisher() accepted an empty broker list")
	}
}

func TestPublisherInterfaceIsExplicit(t *testing.T) {
	var _ Publisher = (*KafkaPublisher)(nil)
	_ = context.Background()
	_ = kafka.Message{}
}
