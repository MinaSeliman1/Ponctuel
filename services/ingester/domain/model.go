package domain

import "time"

// FeedType identifies the GTFS-Realtime feed represented by a snapshot.
type FeedType string

const (
	FeedTypeTripUpdates      FeedType = "trip_updates"
	FeedTypeVehiclePositions FeedType = "vehicle_positions"
)

// EventKind identifies the normalized event family stored by the ingester.
type EventKind string

const (
	EventKindPrediction      EventKind = "prediction"
	EventKindVehiclePosition EventKind = "vehicle_position"
)

type FeedSnapshot struct {
	FeedType        FeedType
	RecordedAt      time.Time
	SourceTimestamp time.Time
	PayloadHash     string
	Entities        []Entity
}

type Entity struct {
	ID              string
	TripUpdate      *TripUpdate
	VehiclePosition *VehiclePosition
}

type TripUpdate struct {
	TripID          string
	RouteID         string
	VehicleID       string
	StopTimeUpdates []StopTimeUpdate
}

type StopTimeUpdate struct {
	StopID       string
	StopSequence uint32
	PredictedAt  time.Time
	DelaySeconds int64
	HasDelay     bool
}

type VehiclePosition struct {
	VehicleID    string
	TripID       string
	RouteID      string
	Latitude     float64
	Longitude    float64
	RecordedAt   time.Time
	DelaySeconds int64
	HasDelay     bool
	HasPosition  bool
}

type Event struct {
	Kind         EventKind
	SnapshotHash string
	FeedType     FeedType
	EntityID     string
	RecordedAt   time.Time
	VehicleID    string
	TripID       string
	RouteID      string
	StopID       string
	StopSequence uint32
	Latitude     float64
	Longitude    float64
	PredictedAt  time.Time
	DelaySeconds int64
	HasDelay     bool
}

type Vehicle struct {
	VehicleID    string
	RouteID      string
	TripID       string
	Latitude     float64
	Longitude    float64
	RecordedAt   time.Time
	DelaySeconds int64
	HasDelay     bool
}
