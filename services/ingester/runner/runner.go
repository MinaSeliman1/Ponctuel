package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"ponctuel/services/bus"
	eventcontract "ponctuel/services/events"
	"ponctuel/services/ingester/config"
	"ponctuel/services/ingester/domain"
	"ponctuel/services/ingester/fetch"
	"ponctuel/services/ingester/store"
)

type Metrics struct {
	fetches          atomic.Uint64
	parseErrors      atomic.Uint64
	duplicates       atomic.Uint64
	insertedEvents   atomic.Uint64
	publishedEvents  atomic.Uint64
	publishErrors    atomic.Uint64
	unauthorized     atomic.Uint64
	rateLimited      atomic.Uint64
	upstreamFailures atomic.Uint64
	httpFailures     atomic.Uint64
}

func NewMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) Fetches() uint64 {
	if m == nil {
		return 0
	}
	return m.fetches.Load()
}

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	if m == nil {
		m = NewMetrics()
	}
	fmt.Fprintf(w, "# HELP ponctuel_ingester_fetch_total Feed fetch attempts.\n# TYPE ponctuel_ingester_fetch_total counter\nponctuel_ingester_fetch_total %d\n", m.fetches.Load())
	fmt.Fprintf(w, "# HELP ponctuel_ingester_parse_errors_total Feed normalization failures.\n# TYPE ponctuel_ingester_parse_errors_total counter\nponctuel_ingester_parse_errors_total %d\n", m.parseErrors.Load())
	fmt.Fprintf(w, "# HELP ponctuel_ingester_duplicate_snapshots_total Known snapshots skipped.\n# TYPE ponctuel_ingester_duplicate_snapshots_total counter\nponctuel_ingester_duplicate_snapshots_total %d\n", m.duplicates.Load())
	fmt.Fprintf(w, "# HELP ponctuel_ingester_inserted_events_total Normalized events inserted.\n# TYPE ponctuel_ingester_inserted_events_total counter\nponctuel_ingester_inserted_events_total %d\n", m.insertedEvents.Load())
	fmt.Fprintf(w, "# HELP ponctuel_ingester_published_events_total Normalized events published to Redpanda.\n# TYPE ponctuel_ingester_published_events_total counter\nponctuel_ingester_published_events_total %d\n", m.publishedEvents.Load())
	fmt.Fprintf(w, "# HELP ponctuel_ingester_publish_errors_total Redpanda publish failures.\n# TYPE ponctuel_ingester_publish_errors_total counter\nponctuel_ingester_publish_errors_total %d\n", m.publishErrors.Load())
	fmt.Fprintf(w, "# HELP ponctuel_ingester_fetch_errors_total Fetch failures by safe status class.\n# TYPE ponctuel_ingester_fetch_errors_total counter\nponctuel_ingester_fetch_errors_total{class=\"unauthorized\"} %d\nponctuel_ingester_fetch_errors_total{class=\"rate_limited\"} %d\nponctuel_ingester_fetch_errors_total{class=\"upstream\"} %d\nponctuel_ingester_fetch_errors_total{class=\"http\"} %d\n", m.unauthorized.Load(), m.rateLimited.Load(), m.upstreamFailures.Load(), m.httpFailures.Load())
}

type Runner struct {
	config     config.Config
	source     fetch.Source
	repository store.Repository
	metrics    *Metrics
	publisher  bus.Publisher
}

func New(cfg config.Config, source fetch.Source, repository store.Repository) *Runner {
	return NewWithMetricsAndPublisher(cfg, source, repository, nil, nil)
}

func NewWithMetricsAndPublisher(cfg config.Config, source fetch.Source, repository store.Repository, metrics *Metrics, publisher bus.Publisher) *Runner {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 30 * time.Second
	}
	if metrics == nil {
		metrics = NewMetrics()
	}
	return &Runner{config: cfg, source: source, repository: repository, metrics: metrics, publisher: publisher}
}

func NewWithMetrics(cfg config.Config, source fetch.Source, repository store.Repository, metrics *Metrics) *Runner {
	return NewWithMetricsAndPublisher(cfg, source, repository, metrics, nil)
}

func Run(ctx context.Context, cfg config.Config, source fetch.Source, repository store.Repository) error {
	return New(cfg, source, repository).Run(ctx)
}

func (r *Runner) Run(ctx context.Context) error {
	if r == nil || r.source == nil || r.repository == nil {
		return fmt.Errorf("runner requires a source and repository")
	}
	if ctx == nil {
		return fmt.Errorf("runner context is nil")
	}

	_ = r.collectOnce(ctx)
	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_ = r.collectOnce(ctx)
		}
	}
}

func (r *Runner) collectOnce(ctx context.Context) error {
	var collectedErrors []error
	for _, feedType := range []domain.FeedType{domain.FeedTypeTripUpdates, domain.FeedTypeVehiclePositions} {
		r.metrics.fetches.Add(1)
		snapshot, err := r.source.Fetch(ctx, feedType)
		if err != nil {
			r.observeFetchError(err)
			collectedErrors = append(collectedErrors, fmt.Errorf("fetch %s: %w", feedType, err))
			continue
		}
		events, err := domain.Normalize(snapshot)
		if err != nil {
			r.metrics.parseErrors.Add(1)
			collectedErrors = append(collectedErrors, fmt.Errorf("normalize %s: %w", feedType, err))
			continue
		}
		snapshot.SourceMode = r.config.AppEnv
		snapshotID, inserted, err := r.repository.InsertSnapshot(ctx, snapshot)
		if err != nil {
			collectedErrors = append(collectedErrors, fmt.Errorf("store %s snapshot: %w", feedType, err))
			continue
		}
		if !inserted {
			r.metrics.duplicates.Add(1)
			continue
		}
		insertedEvents, err := r.repository.InsertEvents(ctx, snapshotID, events)
		if err != nil {
			collectedErrors = append(collectedErrors, fmt.Errorf("store %s events: %w", feedType, err))
			continue
		}
		r.metrics.insertedEvents.Add(uint64(insertedEvents))
		if r.publisher != nil {
			for _, event := range events {
				topic, topicErr := eventcontract.TopicFor(event.Kind)
				if topicErr != nil {
					r.metrics.publishErrors.Add(1)
					collectedErrors = append(collectedErrors, topicErr)
					continue
				}
				if publishErr := r.publisher.Publish(ctx, topic, eventcontract.Key(event), event); publishErr != nil {
					r.metrics.publishErrors.Add(1)
					collectedErrors = append(collectedErrors, fmt.Errorf("publish %s event: %w", topic, publishErr))
					continue
				}
				r.metrics.publishedEvents.Add(1)
			}
		}
	}
	return errors.Join(collectedErrors...)
}

func (r *Runner) observeFetchError(err error) {
	if errors.Is(err, fetch.ErrUnauthorized) {
		r.metrics.unauthorized.Add(1)
		return
	}
	if errors.Is(err, fetch.ErrRateLimited) {
		r.metrics.rateLimited.Add(1)
		return
	}
	if errors.Is(err, fetch.ErrUpstream) {
		r.metrics.upstreamFailures.Add(1)
		return
	}
	var fetchErr *fetch.FetchError
	if errors.As(err, &fetchErr) && fetchErr.Kind == fetch.FetchErrorHTTP {
		r.metrics.httpFailures.Add(1)
	}
}
