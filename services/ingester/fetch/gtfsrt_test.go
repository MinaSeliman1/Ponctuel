package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
	"ponctuel/services/ingester/domain"
)

func TestDecodeTripUpdate(t *testing.T) {
	predictedAt := time.Date(2026, 9, 11, 12, 2, 0, 0, time.UTC)
	payload, err := proto.Marshal(&gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: ptrString("2.0"), Timestamp: ptrUint64(uint64(predictedAt.Unix()))},
		Entity: []*gtfs.FeedEntity{{
			Id: ptrString("entity-1"),
			TripUpdate: &gtfs.TripUpdate{
				Trip:    &gtfs.TripDescriptor{TripId: ptrString("trip-1"), RouteId: ptrString("51")},
				Vehicle: &gtfs.VehicleDescriptor{Id: ptrString("bus-1")},
				StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{
					StopSequence: ptrUint32(7),
					StopId:       ptrString("stop-1"),
					Arrival:      &gtfs.TripUpdate_StopTimeEvent{Time: ptrInt64(predictedAt.Unix()), Delay: ptrInt32(-90)},
				}},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := Decode(payload, predictedAt, domain.FeedTypeTripUpdates)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PayloadHash == "" {
		t.Fatal("Decode returned empty payload hash")
	}
	events, err := domain.Normalize(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].DelaySeconds != -90 {
		t.Fatalf("decoded events = %+v, want one event with delay -90", events)
	}
}

func TestFetcherSendsConfiguredAPIKeyHeader(t *testing.T) {
	var gotHeader string
	predictedAt := time.Date(2026, 9, 11, 12, 2, 0, 0, time.UTC)
	payload, err := proto.Marshal(&gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: ptrString("2.0"), Timestamp: ptrUint64(uint64(predictedAt.Unix()))},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("apikey")
		w.Header().Set("Content-Type", "application/x-google-protobuf")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	fetcher := New(Config{
		Client:              server.Client(),
		TripUpdatesURL:      server.URL,
		VehiclePositionsURL: server.URL,
		APIKey:              "test-key",
		APIKeyHeader:        "apikey",
		MaxPayloadBytes:     1 << 20,
	})
	if _, err := fetcher.Fetch(context.Background(), domain.FeedTypeTripUpdates); err != nil {
		t.Fatal(err)
	}
	if gotHeader != "test-key" {
		t.Fatalf("apikey header = %q, want test-key", gotHeader)
	}
}

func TestDecodeRejectsMalformedProtobuf(t *testing.T) {
	if _, err := Decode([]byte{0xff, 0x00, 0x01}, time.Now().UTC(), domain.FeedTypeTripUpdates); err == nil {
		t.Fatal("Decode returned nil error for malformed protobuf")
	}
}

func TestFetcherFixtureModeUsesDecodePath(t *testing.T) {
	predictedAt := time.Date(2026, 9, 11, 12, 2, 0, 0, time.UTC)
	payload, err := proto.Marshal(&gtfs.FeedMessage{
		Header: &gtfs.FeedHeader{GtfsRealtimeVersion: ptrString("2.0")},
		Entity: []*gtfs.FeedEntity{{
			Id: ptrString("entity-1"),
			TripUpdate: &gtfs.TripUpdate{
				Trip: &gtfs.TripDescriptor{TripId: ptrString("trip-1")},
				StopTimeUpdate: []*gtfs.TripUpdate_StopTimeUpdate{{
					StopId:  ptrString("stop-1"),
					Arrival: &gtfs.TripUpdate_StopTimeEvent{Time: ptrInt64(predictedAt.Unix())},
				}},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(t.TempDir(), "trip_updates.pb")
	if err := os.WriteFile(fixturePath, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := New(Config{
		Mode:                   "fixture",
		TripUpdatesFixturePath: fixturePath,
	}).Fetch(context.Background(), domain.FeedTypeTripUpdates)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PayloadHash != domain.HashPayload(payload) {
		t.Fatalf("fixture hash = %q, want %q", snapshot.PayloadHash, domain.HashPayload(payload))
	}
}

func TestFetcherClassifiesUnauthorizedWithoutLeakingAPIKey(t *testing.T) {
	const secret = "do-not-log-this-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := New(Config{
		Client:         server.Client(),
		TripUpdatesURL: server.URL,
		APIKey:         secret,
	}).Fetch(context.Background(), domain.FeedTypeTripUpdates)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked API key: %v", err)
	}
}

func TestFetcherRejectsOversizedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	_, err := New(Config{
		Client:          server.Client(),
		TripUpdatesURL:  server.URL,
		MaxPayloadBytes: 4,
	}).Fetch(context.Background(), domain.FeedTypeTripUpdates)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("error = %v, want ErrTooLarge", err)
	}
}

func ptrString(value string) *string { return &value }
func ptrUint32(value uint32) *uint32 { return &value }
func ptrUint64(value uint64) *uint64 { return &value }
func ptrInt32(value int32) *int32    { return &value }
func ptrInt64(value int64) *int64    { return &value }
