package matcher

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"ponctuel/services/ingester/domain"
)

const defaultGeofenceRadiusMeters = 60.0

type Engine struct {
	mu              sync.Mutex
	radiusMeters    float64
	predictions     map[string]domain.Event
	emittedArrivals map[string]struct{}
}

func NewEngine(radiusMeters float64) *Engine {
	if radiusMeters <= 0 {
		radiusMeters = defaultGeofenceRadiusMeters
	}
	return &Engine{
		radiusMeters:    radiusMeters,
		predictions:     make(map[string]domain.Event),
		emittedArrivals: make(map[string]struct{}),
	}
}

func (e *Engine) HandleEvent(event domain.Event, stops []domain.Stop) ([]domain.ArrivalObserved, error) {
	if e == nil {
		return nil, fmt.Errorf("matcher engine is nil")
	}
	if event.RecordedAt.IsZero() {
		return nil, fmt.Errorf("event recorded_at is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch event.Kind {
	case domain.EventKindPrediction:
		return e.handlePrediction(event), nil
	case domain.EventKindVehiclePosition:
		return e.handleVehicle(event, stops), nil
	default:
		return nil, fmt.Errorf("unsupported event kind %q", event.Kind)
	}
}

func (e *Engine) handlePrediction(event domain.Event) []domain.ArrivalObserved {
	if strings.TrimSpace(event.TripID) == "" || strings.TrimSpace(event.StopID) == "" || event.PredictedAt.IsZero() {
		return nil
	}
	key := predictionKey(event)
	e.predictions[key] = event
	if event.PredictedAt.After(event.RecordedAt) {
		return nil
	}
	return e.emit(domain.ArrivalObserved{
		TripID:       event.TripID,
		ServiceDate:  serviceDate(event.PredictedAt),
		StopID:       event.StopID,
		StopSequence: event.StopSequence,
		ObservedAt:   event.RecordedAt.UTC(),
		Method:       domain.ArrivalMethodLastUpdate,
		Confidence:   0.70,
		Reason:       "prediction was already due when consumed",
	})
}

func (e *Engine) handleVehicle(event domain.Event, stops []domain.Stop) []domain.ArrivalObserved {
	if strings.TrimSpace(event.TripID) == "" || !validCoordinate(event.Latitude, event.Longitude) {
		return nil
	}
	stopCoordinates := make(map[string]domain.Stop, len(stops))
	for _, stop := range stops {
		if validCoordinate(stop.Latitude, stop.Longitude) && stop.StopID != "" {
			stopCoordinates[stop.StopID] = stop
		}
	}

	arrivals := make([]domain.ArrivalObserved, 0)
	for _, prediction := range e.predictions {
		if prediction.TripID != event.TripID {
			continue
		}
		stop, ok := stopCoordinates[prediction.StopID]
		if !ok {
			continue
		}
		distance := haversineMeters(event.Latitude, event.Longitude, stop.Latitude, stop.Longitude)
		if distance > e.radiusMeters {
			continue
		}
		confidence := 1 - (distance/e.radiusMeters)*0.5
		arrivals = append(arrivals, e.emit(domain.ArrivalObserved{
			TripID:       prediction.TripID,
			ServiceDate:  serviceDate(prediction.PredictedAt),
			StopID:       prediction.StopID,
			StopSequence: prediction.StopSequence,
			ObservedAt:   event.RecordedAt.UTC(),
			Method:       domain.ArrivalMethodGeofence,
			Confidence:   confidence,
			Reason:       fmt.Sprintf("vehicle entered stop geofence at %.1fm", distance),
		})...)
	}
	return arrivals
}

func (e *Engine) emit(arrival domain.ArrivalObserved) []domain.ArrivalObserved {
	key := strings.Join([]string{arrival.TripID, arrival.ServiceDate, arrival.StopID, string(arrival.Method)}, "/")
	if _, exists := e.emittedArrivals[key]; exists {
		return nil
	}
	e.emittedArrivals[key] = struct{}{}
	return []domain.ArrivalObserved{arrival}
}

func predictionKey(event domain.Event) string {
	return strings.Join([]string{event.TripID, serviceDate(event.PredictedAt), event.StopID}, "/")
}

func serviceDate(value time.Time) string {
	return value.UTC().Format("2006-01-02")
}

func validCoordinate(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsInf(latitude, 0) && latitude >= -90 && latitude <= 90 &&
		!math.IsNaN(longitude) && !math.IsInf(longitude, 0) && longitude >= -180 && longitude <= 180
}

func haversineMeters(latitudeA, longitudeA, latitudeB, longitudeB float64) float64 {
	const earthRadiusMeters = 6371000.0
	latitudeDelta := (latitudeB - latitudeA) * math.Pi / 180
	longitudeDelta := (longitudeB - longitudeA) * math.Pi / 180
	latitudeARadians := latitudeA * math.Pi / 180
	latitudeBRadians := latitudeB * math.Pi / 180
	a := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) +
		math.Cos(latitudeARadians)*math.Cos(latitudeBRadians)*math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	return 2 * earthRadiusMeters * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
