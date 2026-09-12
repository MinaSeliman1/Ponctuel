package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

func HashPayload(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func Normalize(snapshot FeedSnapshot) ([]Event, error) {
	if snapshot.RecordedAt.IsZero() {
		return nil, fmt.Errorf("snapshot recorded_at is required")
	}
	if snapshot.FeedType != FeedTypeTripUpdates && snapshot.FeedType != FeedTypeVehiclePositions {
		return nil, fmt.Errorf("unsupported feed type %q", snapshot.FeedType)
	}

	events := make([]Event, 0, len(snapshot.Entities))
	for index, entity := range snapshot.Entities {
		if entity.TripUpdate == nil && entity.VehiclePosition == nil {
			continue
		}
		if strings.TrimSpace(entity.ID) == "" {
			return nil, fmt.Errorf("entity %d is missing an id", index)
		}

		switch snapshot.FeedType {
		case FeedTypeTripUpdates:
			if entity.TripUpdate == nil {
				continue
			}
			predictionEvents, err := normalizeTripUpdate(snapshot, entity)
			if err != nil {
				return nil, err
			}
			events = append(events, predictionEvents...)
		case FeedTypeVehiclePositions:
			if entity.VehiclePosition == nil {
				continue
			}
			positionEvent, ok, err := normalizeVehiclePosition(snapshot, entity)
			if err != nil {
				return nil, err
			}
			if ok {
				events = append(events, positionEvent)
			}
		}
	}

	return events, nil
}

func normalizeTripUpdate(snapshot FeedSnapshot, entity Entity) ([]Event, error) {
	trip := entity.TripUpdate
	if strings.TrimSpace(trip.TripID) == "" {
		return nil, fmt.Errorf("entity %q trip_id is required", entity.ID)
	}

	events := make([]Event, 0, len(trip.StopTimeUpdates))
	for index, stop := range trip.StopTimeUpdates {
		if strings.TrimSpace(stop.StopID) == "" {
			return nil, fmt.Errorf("entity %q stop update %d stop_id is required", entity.ID, index)
		}
		predictedAt := stop.PredictedAt
		if predictedAt.IsZero() {
			predictedAt = snapshot.SourceTimestamp
		}
		if predictedAt.IsZero() {
			predictedAt = snapshot.RecordedAt
		}
		if predictedAt.IsZero() {
			return nil, fmt.Errorf("entity %q stop %q predicted_at is required", entity.ID, stop.StopID)
		}

		events = append(events, Event{
			Kind:         EventKindPrediction,
			SnapshotHash: snapshot.PayloadHash,
			FeedType:     snapshot.FeedType,
			EntityID:     entity.ID,
			RecordedAt:   snapshot.RecordedAt,
			VehicleID:    trip.VehicleID,
			TripID:       trip.TripID,
			RouteID:      trip.RouteID,
			StopID:       stop.StopID,
			StopSequence: stop.StopSequence,
			PredictedAt:  predictedAt,
			DelaySeconds: stop.DelaySeconds,
			HasDelay:     stop.HasDelay,
		})
	}
	return events, nil
}

func normalizeVehiclePosition(snapshot FeedSnapshot, entity Entity) (Event, bool, error) {
	position := entity.VehiclePosition
	if strings.TrimSpace(position.VehicleID) == "" {
		return Event{}, false, fmt.Errorf("entity %q vehicle_id is required", entity.ID)
	}

	// HasPosition is authoritative for decoded protobufs. The non-zero fallback
	// keeps hand-built domain fixtures concise while retaining valid 0,0 data
	// when the decoder explicitly sets HasPosition.
	hasPosition := position.HasPosition || position.Latitude != 0 || position.Longitude != 0
	if !hasPosition {
		return Event{}, false, nil
	}
	if math.IsNaN(position.Latitude) || math.IsInf(position.Latitude, 0) || position.Latitude < -90 || position.Latitude > 90 {
		return Event{}, false, fmt.Errorf("entity %q latitude is invalid", entity.ID)
	}
	if math.IsNaN(position.Longitude) || math.IsInf(position.Longitude, 0) || position.Longitude < -180 || position.Longitude > 180 {
		return Event{}, false, fmt.Errorf("entity %q longitude is invalid", entity.ID)
	}

	recordedAt := position.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = snapshot.RecordedAt
	}
	if recordedAt.IsZero() {
		return Event{}, false, fmt.Errorf("entity %q recorded_at is required", entity.ID)
	}

	return Event{
		Kind:         EventKindVehiclePosition,
		SnapshotHash: snapshot.PayloadHash,
		FeedType:     snapshot.FeedType,
		EntityID:     entity.ID,
		RecordedAt:   recordedAt,
		VehicleID:    position.VehicleID,
		TripID:       position.TripID,
		RouteID:      position.RouteID,
		Latitude:     position.Latitude,
		Longitude:    position.Longitude,
		DelaySeconds: position.DelaySeconds,
		HasDelay:     position.HasDelay,
	}, true, nil
}
