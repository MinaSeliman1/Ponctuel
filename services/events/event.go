package events

import (
	"encoding/json"
	"fmt"
	"strings"

	"ponctuel/services/ingester/domain"
)

const SchemaVersion = 1

const (
	TopicTripUpdates      = "trip-updates"
	TopicVehiclePositions = "vehicle-positions"
)

type Message struct {
	SchemaVersion int          `json:"schema_version"`
	Event         domain.Event `json:"event"`
}

func Encode(event domain.Event) ([]byte, error) {
	if err := validate(event); err != nil {
		return nil, err
	}
	return json.Marshal(Message{SchemaVersion: SchemaVersion, Event: event})
}

func Decode(payload []byte) (domain.Event, error) {
	var message Message
	if err := json.Unmarshal(payload, &message); err != nil {
		return domain.Event{}, fmt.Errorf("decode event JSON: %w", err)
	}
	if message.SchemaVersion != SchemaVersion {
		return domain.Event{}, fmt.Errorf("unsupported event schema version %d", message.SchemaVersion)
	}
	if err := validate(message.Event); err != nil {
		return domain.Event{}, err
	}
	return message.Event, nil
}

func TopicFor(kind domain.EventKind) (string, error) {
	switch kind {
	case domain.EventKindPrediction:
		return TopicTripUpdates, nil
	case domain.EventKindVehiclePosition:
		return TopicVehiclePositions, nil
	default:
		return "", fmt.Errorf("unsupported event kind %q", kind)
	}
}

func Key(event domain.Event) string {
	return strings.Join([]string{event.SnapshotHash, event.EntityID, event.StopID, event.VehicleID}, "/")
}

func validate(event domain.Event) error {
	if event.RecordedAt.IsZero() {
		return fmt.Errorf("event recorded_at is required")
	}
	switch event.Kind {
	case domain.EventKindPrediction:
		if strings.TrimSpace(event.TripID) == "" || strings.TrimSpace(event.StopID) == "" {
			return fmt.Errorf("prediction event trip_id and stop_id are required")
		}
		if event.PredictedAt.IsZero() {
			return fmt.Errorf("prediction event predicted_at is required")
		}
	case domain.EventKindVehiclePosition:
		if strings.TrimSpace(event.VehicleID) == "" {
			return fmt.Errorf("vehicle position event vehicle_id is required")
		}
	default:
		return fmt.Errorf("unsupported event kind %q", event.Kind)
	}
	return nil
}
