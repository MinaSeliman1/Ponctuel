package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
	"ponctuel/services/ingester/domain"
)

const defaultMaxPayloadBytes int64 = 16 << 20

var (
	ErrUnauthorized = errors.New("upstream authorization failed")
	ErrRateLimited  = errors.New("upstream rate limit reached")
	ErrUpstream     = errors.New("upstream service failed")
	ErrMalformed    = errors.New("malformed GTFS-Realtime payload")
	ErrTooLarge     = errors.New("GTFS-Realtime payload is too large")
)

type FetchErrorKind string

const (
	FetchErrorUnauthorized FetchErrorKind = "unauthorized"
	FetchErrorRateLimited  FetchErrorKind = "rate_limited"
	FetchErrorUpstream     FetchErrorKind = "upstream"
	FetchErrorHTTP         FetchErrorKind = "http"
	FetchErrorMalformed    FetchErrorKind = "malformed"
	FetchErrorTooLarge     FetchErrorKind = "too_large"
)

type FetchError struct {
	Kind       FetchErrorKind
	StatusCode int
	Err        error
}

// Source fetches one GTFS-Realtime snapshot for the requested feed.
type Source interface {
	Fetch(ctx context.Context, feedType domain.FeedType) (domain.FeedSnapshot, error)
}

func (e *FetchError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("fetch %s: HTTP %d", e.Kind, e.StatusCode)
	}
	if e.Err != nil {
		return fmt.Sprintf("fetch %s: %v", e.Kind, e.Err)
	}
	return fmt.Sprintf("fetch %s", e.Kind)
}

func (e *FetchError) Unwrap() error { return e.Err }

func (e *FetchError) Is(target error) bool {
	switch e.Kind {
	case FetchErrorUnauthorized:
		return target == ErrUnauthorized
	case FetchErrorRateLimited:
		return target == ErrRateLimited
	case FetchErrorUpstream:
		return target == ErrUpstream
	case FetchErrorMalformed:
		return target == ErrMalformed
	case FetchErrorTooLarge:
		return target == ErrTooLarge
	default:
		return false
	}
}

type Config struct {
	Client                      *http.Client
	Mode                        string
	FixtureDir                  string
	TripUpdatesFixturePath      string
	VehiclePositionsFixturePath string
	TripUpdatesURL              string
	VehiclePositionsURL         string
	APIKey                      string
	APIKeyHeader                string
	MaxPayloadBytes             int64
	Timeout                     time.Duration
}

type Fetcher struct {
	config Config
}

func New(config Config) *Fetcher {
	if config.APIKeyHeader == "" {
		config.APIKeyHeader = "apikey"
	}
	if config.MaxPayloadBytes <= 0 {
		config.MaxPayloadBytes = defaultMaxPayloadBytes
	}
	return &Fetcher{config: config}
}

func (f *Fetcher) Fetch(ctx context.Context, feedType domain.FeedType) (domain.FeedSnapshot, error) {
	if f == nil {
		return domain.FeedSnapshot{}, fmt.Errorf("fetcher is nil")
	}
	if f.config.Mode == "fixture" {
		path, err := f.fixturePath(feedType)
		if err != nil {
			return domain.FeedSnapshot{}, err
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return domain.FeedSnapshot{}, fmt.Errorf("read fixture %q: %w", filepath.Base(path), err)
		}
		return Decode(payload, time.Now().UTC(), feedType)
	}

	endpoint, err := f.endpoint(feedType)
	if err != nil {
		return domain.FeedSnapshot{}, err
	}
	requestContext := ctx
	if f.config.Timeout > 0 {
		var cancel context.CancelFunc
		requestContext, cancel = context.WithTimeout(ctx, f.config.Timeout)
		defer cancel()
	}
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if err != nil {
		return domain.FeedSnapshot{}, fmt.Errorf("build feed request: %w", err)
	}
	request.Header.Set("Accept", "application/x-google-protobuf")
	if f.config.APIKey != "" {
		request.Header.Set(f.config.APIKeyHeader, f.config.APIKey)
	}

	client := f.config.Client
	if client == nil {
		client = &http.Client{}
	}
	response, err := client.Do(request)
	if err != nil {
		return domain.FeedSnapshot{}, fmt.Errorf("request feed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return domain.FeedSnapshot{}, classifyStatus(response.StatusCode)
	}

	payload, err := io.ReadAll(io.LimitReader(response.Body, f.config.MaxPayloadBytes+1))
	if err != nil {
		return domain.FeedSnapshot{}, fmt.Errorf("read feed response: %w", err)
	}
	if int64(len(payload)) > f.config.MaxPayloadBytes {
		return domain.FeedSnapshot{}, &FetchError{Kind: FetchErrorTooLarge, Err: ErrTooLarge}
	}
	return Decode(payload, time.Now().UTC(), feedType)
}

func Decode(payload []byte, recordedAt time.Time, feedType domain.FeedType) (domain.FeedSnapshot, error) {
	if feedType != domain.FeedTypeTripUpdates && feedType != domain.FeedTypeVehiclePositions {
		return domain.FeedSnapshot{}, fmt.Errorf("unsupported feed type %q", feedType)
	}
	message := &gtfs.FeedMessage{}
	if err := proto.Unmarshal(payload, message); err != nil {
		return domain.FeedSnapshot{}, &FetchError{Kind: FetchErrorMalformed, Err: fmt.Errorf("%w: %v", ErrMalformed, err)}
	}
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	snapshot := domain.FeedSnapshot{
		FeedType:        feedType,
		RecordedAt:      recordedAt.UTC(),
		SourceTimestamp: fromUnixSeconds(int64(message.GetHeader().GetTimestamp())),
		PayloadHash:     domain.HashPayload(payload),
		Entities:        make([]domain.Entity, 0, len(message.GetEntity())),
	}
	for _, entity := range message.GetEntity() {
		if entity == nil {
			continue
		}
		converted := domain.Entity{ID: entity.GetId()}
		switch feedType {
		case domain.FeedTypeTripUpdates:
			converted.TripUpdate = decodeTripUpdate(entity.GetTripUpdate(), snapshot.SourceTimestamp, snapshot.RecordedAt)
		case domain.FeedTypeVehiclePositions:
			converted.VehiclePosition = decodeVehiclePosition(entity.GetVehicle(), snapshot.RecordedAt)
		}
		snapshot.Entities = append(snapshot.Entities, converted)
	}
	return snapshot, nil
}

func decodeTripUpdate(update *gtfs.TripUpdate, sourceTimestamp, recordedAt time.Time) *domain.TripUpdate {
	if update == nil {
		return nil
	}
	trip := update.GetTrip()
	vehicle := update.GetVehicle()
	converted := &domain.TripUpdate{
		TripID:    trip.GetTripId(),
		RouteID:   trip.GetRouteId(),
		VehicleID: vehicle.GetId(),
	}
	for _, stop := range update.GetStopTimeUpdate() {
		if stop == nil {
			continue
		}
		convertedStop := domain.StopTimeUpdate{
			StopID:       stop.GetStopId(),
			StopSequence: stop.GetStopSequence(),
		}
		if arrival := stop.GetArrival(); arrival != nil {
			convertedStop.PredictedAt = fromUnixSeconds(arrival.GetTime())
			convertedStop.DelaySeconds = int64(arrival.GetDelay())
			convertedStop.HasDelay = arrival.Delay != nil
		}
		if convertedStop.PredictedAt.IsZero() {
			if departure := stop.GetDeparture(); departure != nil {
				convertedStop.PredictedAt = fromUnixSeconds(departure.GetTime())
				if !convertedStop.HasDelay {
					convertedStop.DelaySeconds = int64(departure.GetDelay())
					convertedStop.HasDelay = departure.Delay != nil
				}
			}
		}
		if convertedStop.PredictedAt.IsZero() {
			convertedStop.PredictedAt = sourceTimestamp
		}
		if convertedStop.PredictedAt.IsZero() {
			convertedStop.PredictedAt = recordedAt
		}
		converted.StopTimeUpdates = append(converted.StopTimeUpdates, convertedStop)
	}
	return converted
}

func decodeVehiclePosition(vehicle *gtfs.VehiclePosition, recordedAt time.Time) *domain.VehiclePosition {
	if vehicle == nil {
		return nil
	}
	trip := vehicle.GetTrip()
	converted := &domain.VehiclePosition{
		VehicleID:  vehicle.GetVehicle().GetId(),
		TripID:     trip.GetTripId(),
		RouteID:    trip.GetRouteId(),
		RecordedAt: recordedAt,
	}
	if timestamp := vehicle.GetTimestamp(); timestamp != 0 {
		converted.RecordedAt = fromUnixSeconds(int64(timestamp))
	}
	if position := vehicle.GetPosition(); position != nil {
		converted.Latitude = float64(position.GetLatitude())
		converted.Longitude = float64(position.GetLongitude())
		converted.HasPosition = true
	}
	return converted
}

func (f *Fetcher) endpoint(feedType domain.FeedType) (string, error) {
	switch feedType {
	case domain.FeedTypeTripUpdates:
		if strings.TrimSpace(f.config.TripUpdatesURL) == "" {
			return "", fmt.Errorf("trip updates URL is required")
		}
		return f.config.TripUpdatesURL, nil
	case domain.FeedTypeVehiclePositions:
		if strings.TrimSpace(f.config.VehiclePositionsURL) == "" {
			return "", fmt.Errorf("vehicle positions URL is required")
		}
		return f.config.VehiclePositionsURL, nil
	default:
		return "", fmt.Errorf("unsupported feed type %q", feedType)
	}
}

func (f *Fetcher) fixturePath(feedType domain.FeedType) (string, error) {
	switch feedType {
	case domain.FeedTypeTripUpdates:
		if f.config.TripUpdatesFixturePath != "" {
			return f.config.TripUpdatesFixturePath, nil
		}
		if f.config.FixtureDir != "" {
			return filepath.Join(f.config.FixtureDir, "trip_updates.pb"), nil
		}
	case domain.FeedTypeVehiclePositions:
		if f.config.VehiclePositionsFixturePath != "" {
			return f.config.VehiclePositionsFixturePath, nil
		}
		if f.config.FixtureDir != "" {
			return filepath.Join(f.config.FixtureDir, "vehicle_positions.pb"), nil
		}
	default:
		return "", fmt.Errorf("unsupported feed type %q", feedType)
	}
	return "", fmt.Errorf("fixture path is required for %q", feedType)
}

func classifyStatus(statusCode int) error {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &FetchError{Kind: FetchErrorUnauthorized, StatusCode: statusCode, Err: ErrUnauthorized}
	case http.StatusTooManyRequests:
		return &FetchError{Kind: FetchErrorRateLimited, StatusCode: statusCode, Err: ErrRateLimited}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return &FetchError{Kind: FetchErrorUpstream, StatusCode: statusCode, Err: ErrUpstream}
	default:
		return &FetchError{Kind: FetchErrorHTTP, StatusCode: statusCode}
	}
}

func fromUnixSeconds(seconds int64) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(seconds), 0).UTC()
}
