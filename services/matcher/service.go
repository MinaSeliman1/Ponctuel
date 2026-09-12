package matcher

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"ponctuel/services/bus"
	"ponctuel/services/events"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/domain"
	"ponctuel/services/ingester/store"
)

type Service struct {
	config  config.Config
	repo    store.Repository
	readers []bus.Reader
	engine  *Engine
	metrics *Metrics
	ready   atomic.Bool
	stops   []domain.Stop
}

func NewService(cfg config.Config, repository store.Repository, readers []bus.Reader, metrics *Metrics) *Service {
	if metrics == nil {
		metrics = NewMetrics()
	}
	return &Service{
		config:  cfg,
		repo:    repository,
		readers: readers,
		engine:  NewEngine(cfg.GeofenceRadiusMeters),
		metrics: metrics,
	}
}

func (s *Service) Metrics() *Metrics { return s.metrics }

func (s *Service) Ready() bool {
	return s != nil && s.ready.Load()
}

func (s *Service) Run(ctx context.Context) error {
	if s == nil || s.repo == nil || len(s.readers) == 0 {
		return fmt.Errorf("matcher requires repository and Kafka readers")
	}
	if ctx == nil {
		return fmt.Errorf("matcher context is nil")
	}
	stops, err := s.repo.LatestStops(ctx)
	if err != nil {
		return fmt.Errorf("load GTFS stops: %w", err)
	}
	if len(stops) == 0 && s.config.MatcherStopsFile != "" {
		stops, err = loadStopsFile(s.config.MatcherStopsFile)
		if err != nil {
			return err
		}
	}
	s.stops = stops
	if err := s.repo.Ping(ctx); err != nil {
		return fmt.Errorf("matcher database readiness: %w", err)
	}
	s.ready.Store(true)

	runContext, cancel := context.WithCancel(ctx)
	defer cancel()
	errors := make(chan error, len(s.readers))
	for _, reader := range s.readers {
		go func(reader bus.Reader) { errors <- s.consume(runContext, reader) }(reader)
	}
	select {
	case <-ctx.Done():
		return nil
	case err := <-errors:
		return err
	}
}

func (s *Service) Close() error {
	var firstErr error
	for _, reader := range s.readers {
		if err := reader.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Service) consume(ctx context.Context, reader bus.Reader) error {
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch matcher message: %w", err)
		}
		s.metrics.events.Add(1)
		event, err := events.Decode(message.Value)
		if err != nil {
			s.metrics.errors.Add(1)
			if commitErr := reader.CommitMessages(ctx, message); commitErr != nil {
				return fmt.Errorf("commit invalid matcher message: %w", commitErr)
			}
			continue
		}
		arrivals, err := s.engine.HandleEvent(event, s.stops)
		if err != nil {
			s.metrics.errors.Add(1)
			return err
		}
		for _, arrival := range arrivals {
			inserted, insertErr := s.repo.InsertArrival(ctx, arrival)
			if insertErr != nil {
				s.metrics.errors.Add(1)
				return fmt.Errorf("store matcher arrival: %w", insertErr)
			}
			if inserted {
				s.metrics.arrivals.Add(1)
			} else {
				s.metrics.duplicates.Add(1)
			}
		}
		if err := reader.CommitMessages(ctx, message); err != nil {
			return fmt.Errorf("commit matcher message: %w", err)
		}
	}
}

func loadStopsFile(path string) ([]domain.Stop, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open matcher stops file: %w", err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read matcher stops header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for index, column := range header {
		columns[strings.ToLower(strings.TrimSpace(column))] = index
	}
	for _, required := range []string{"stop_id", "stop_lat", "stop_lon"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("matcher stops file is missing %s", required)
		}
	}
	stops := make([]domain.Stop, 0)
	for {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read matcher stop: %w", readErr)
		}
		stopID := strings.TrimSpace(row[columns["stop_id"]])
		latitude, latitudeErr := strconv.ParseFloat(strings.TrimSpace(row[columns["stop_lat"]]), 64)
		longitude, longitudeErr := strconv.ParseFloat(strings.TrimSpace(row[columns["stop_lon"]]), 64)
		if stopID == "" || latitudeErr != nil || longitudeErr != nil {
			return nil, fmt.Errorf("invalid matcher stop row for %q", stopID)
		}
		stops = append(stops, domain.Stop{StopID: stopID, Latitude: latitude, Longitude: longitude})
	}
	return stops, nil
}
