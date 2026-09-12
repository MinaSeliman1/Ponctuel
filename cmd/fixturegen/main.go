package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

func main() {
	root := "testdata/realtime"
	if err := os.MkdirAll(root, 0o755); err != nil {
		fatal(err)
	}

	if err := writeFeed(filepath.Join(root, "vehicle_positions.pb"), vehiclePositionsFeed()); err != nil {
		fatal(err)
	}
	if err := writeFeed(filepath.Join(root, "trip_updates.pb"), tripUpdatesFeed()); err != nil {
		fatal(err)
	}
}

func writeFeed(path string, feed *gtfs.FeedMessage) error {
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(feed)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func vehiclePositionsFeed() *gtfs.FeedMessage {
	recordedAt := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC).Unix()
	return &gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: ptrString("2.0"), Timestamp: ptrUint64(uint64(recordedAt))},
		Entity: []*gtfs.FeedEntity{{
			Id: ptrString("vehicle-entity-1"),
			Vehicle: &gtfs.VehiclePosition{
				Trip:      &gtfs.TripDescriptor{TripId: ptrString("trip-1"), RouteId: ptrString("51")},
				Vehicle:   &gtfs.VehicleDescriptor{Id: ptrString("bus-1")},
				Position:  &gtfs.Position{Latitude: ptrFloat32(45.5017), Longitude: ptrFloat32(-73.5673)},
				Timestamp: ptrUint64(uint64(recordedAt)),
			},
		}},
	}
}

func tripUpdatesFeed() *gtfs.FeedMessage {
	predictedAt := time.Date(2026, 9, 11, 12, 2, 0, 0, time.UTC).Unix()
	return &gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: ptrString("2.0"), Timestamp: ptrUint64(uint64(predictedAt))},
		Entity: []*gtfs.FeedEntity{{
			Id: ptrString("trip-entity-1"),
			TripUpdate: &gtfs.TripUpdate{
				Trip:    &gtfs.TripDescriptor{TripId: ptrString("trip-1"), RouteId: ptrString("51")},
				Vehicle: &gtfs.VehicleDescriptor{Id: ptrString("bus-1")},
				StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{
					StopSequence: ptrUint32(7),
					StopId:       ptrString("stop-1"),
					Arrival:      &gtfs.TripUpdate_StopTimeEvent{Time: ptrInt64(predictedAt), Delay: ptrInt32(-90)},
				}},
			},
		}},
	}
}

func ptrString(value string) *string    { return &value }
func ptrUint32(value uint32) *uint32    { return &value }
func ptrUint64(value uint64) *uint64    { return &value }
func ptrInt32(value int32) *int32       { return &value }
func ptrInt64(value int64) *int64       { return &value }
func ptrFloat32(value float32) *float32 { return &value }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
