package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"ponctuel/services/ingester/domain"
)

const defaultFreshnessWindow = 2 * time.Minute

type Repository interface {
	InsertSnapshot(ctx context.Context, snapshot domain.FeedSnapshot) (snapshotID int64, inserted bool, err error)
	InsertEvents(ctx context.Context, snapshotID int64, events []domain.Event) (int, error)
	CountEvents(ctx context.Context) (int64, error)
	LatestVehicles(ctx context.Context) ([]domain.Vehicle, error)
	InsertArrival(ctx context.Context, arrival domain.ArrivalObserved) (bool, error)
	LatestStops(ctx context.Context) ([]domain.Stop, error)
	ErrorSummary(ctx context.Context, limit int) ([]domain.ErrorSummary, error)
	Ping(ctx context.Context) error
}

type PostgresRepository struct {
	pool            *pgxpool.Pool
	freshnessWindow time.Duration
	sourceMode      string
}

func NewPostgresRepository(pool *pgxpool.Pool, freshnessWindow time.Duration) *PostgresRepository {
	return NewPostgresRepositoryWithSourceMode(pool, freshnessWindow, "fixture")
}

func NewPostgresRepositoryWithSourceMode(pool *pgxpool.Pool, freshnessWindow time.Duration, sourceMode string) *PostgresRepository {
	if freshnessWindow <= 0 {
		freshnessWindow = defaultFreshnessWindow
	}
	mode := strings.ToLower(strings.TrimSpace(sourceMode))
	if mode != "stm" {
		mode = "fixture"
	}
	return &PostgresRepository{pool: pool, freshnessWindow: freshnessWindow, sourceMode: mode}
}

func Connect(ctx context.Context, databaseURL string, freshnessWindow time.Duration) (*PostgresRepository, error) {
	return ConnectWithSourceMode(ctx, databaseURL, freshnessWindow, "fixture")
}

func ConnectWithSourceMode(ctx context.Context, databaseURL string, freshnessWindow time.Duration, sourceMode string) (*PostgresRepository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return NewPostgresRepositoryWithSourceMode(pool, freshnessWindow, sourceMode), nil
}

func (r *PostgresRepository) Close() {
	if r != nil && r.pool != nil {
		r.pool.Close()
	}
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("PostgreSQL repository is not initialized")
	}
	if err := r.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return nil
}

func (r *PostgresRepository) InsertSnapshot(ctx context.Context, snapshot domain.FeedSnapshot) (int64, bool, error) {
	if r == nil || r.pool == nil {
		return 0, false, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	if snapshot.RecordedAt.IsZero() {
		return 0, false, fmt.Errorf("insert snapshot: recorded_at is required")
	}
	if snapshot.PayloadHash == "" {
		return 0, false, fmt.Errorf("insert snapshot: payload_hash is required")
	}
	sourceMode := strings.ToLower(strings.TrimSpace(snapshot.SourceMode))
	if sourceMode == "" {
		sourceMode = r.sourceMode
	}
	if sourceMode != "stm" && sourceMode != "fixture" {
		return 0, false, fmt.Errorf("insert snapshot: unsupported source mode %q", sourceMode)
	}

	var sourceTimestamp any
	if !snapshot.SourceTimestamp.IsZero() {
		sourceTimestamp = snapshot.SourceTimestamp
	}
	var snapshotID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO feed_snapshot (feed_type, source_mode, recorded_at, source_timestamp, payload_hash)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (feed_type, payload_hash) DO NOTHING
		RETURNING snapshot_id
	`, string(snapshot.FeedType), sourceMode, snapshot.RecordedAt, sourceTimestamp, snapshot.PayloadHash).Scan(&snapshotID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("insert feed snapshot: %w", err)
	}
	return snapshotID, true, nil
}

func (r *PostgresRepository) InsertEvents(ctx context.Context, snapshotID int64, events []domain.Event) (int, error) {
	if r == nil || r.pool == nil {
		return 0, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	if snapshotID <= 0 {
		return 0, fmt.Errorf("insert events: snapshot_id must be positive")
	}
	if len(events) == 0 {
		return 0, nil
	}

	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin event transaction: %w", err)
	}
	defer transaction.Rollback(ctx)

	inserted := 0
	for index, event := range events {
		if event.RecordedAt.IsZero() {
			return 0, fmt.Errorf("insert event %d: recorded_at is required", index)
		}
		delay := optionalDelay(event.HasDelay, event.DelaySeconds)
		switch event.Kind {
		case domain.EventKindVehiclePosition:
			if event.VehicleID == "" {
				return 0, fmt.Errorf("insert event %d: vehicle_id is required", index)
			}
			_, err = transaction.Exec(ctx, `
				INSERT INTO vehicle_position
					(snapshot_id, entity_id, recorded_at, vehicle_id, trip_id, route_id,
					 latitude, longitude, delay_seconds)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, snapshotID, event.EntityID, event.RecordedAt, event.VehicleID, nullableString(event.TripID), nullableString(event.RouteID), event.Latitude, event.Longitude, delay)
		case domain.EventKindPrediction:
			if event.TripID == "" {
				return 0, fmt.Errorf("insert event %d: trip_id is required", index)
			}
			if event.StopID == "" {
				return 0, fmt.Errorf("insert event %d: stop_id is required", index)
			}
			predictedAt := event.PredictedAt
			if predictedAt.IsZero() {
				return 0, fmt.Errorf("insert event %d: predicted_at is required", index)
			}
			if !event.HasDelay {
				derivedDelay, deriveErr := deriveScheduleDelay(ctx, transaction, event, predictedAt)
				if deriveErr != nil {
					return 0, fmt.Errorf("derive event %d schedule delay: %w", index, deriveErr)
				}
				delay = derivedDelay
			}
			_, err = transaction.Exec(ctx, `
				INSERT INTO prediction
					(snapshot_id, entity_id, recorded_at, service_date, vehicle_id, trip_id,
					 route_id, stop_id, stop_sequence, predicted_at, delay_seconds)
				VALUES ($1, $2, $3, (($9::timestamptz AT TIME ZONE 'America/Toronto')::date), $4, $5, $6, $7, $8, $9, $10)
			`, snapshotID, event.EntityID, event.RecordedAt, nullableString(event.VehicleID), event.TripID, nullableString(event.RouteID), event.StopID, event.StopSequence, predictedAt, delay)
		default:
			return 0, fmt.Errorf("insert event %d: unsupported kind %q", index, event.Kind)
		}
		if err != nil {
			return 0, fmt.Errorf("insert event %d: %w", index, err)
		}
		inserted++
	}
	if err := transaction.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit event transaction: %w", err)
	}
	return inserted, nil
}

func deriveScheduleDelay(ctx context.Context, transaction pgx.Tx, event domain.Event, predictedAt time.Time) (any, error) {
	var delay pgtype.Int4
	err := transaction.QueryRow(ctx, `
		WITH scheduled AS (
			SELECT
				(
					(
						(($1::timestamptz AT TIME ZONE 'America/Toronto')::date
						 + COALESCE(stop_time.arrival_seconds, stop_time.departure_seconds) * INTERVAL '1 second')
						AT TIME ZONE 'America/Toronto'
					)
				) AS scheduled_at
			FROM gtfs_stop_time AS stop_time
			JOIN gtfs_feed_version AS feed_version
			  ON feed_version.feed_version = stop_time.feed_version
			WHERE stop_time.trip_id = $2
			  AND stop_time.stop_id = $3
			  AND ($4::integer = 0 OR stop_time.stop_sequence = $4::integer)
			  AND COALESCE(stop_time.arrival_seconds, stop_time.departure_seconds) IS NOT NULL
			ORDER BY feed_version.retrieved_at DESC, feed_version.feed_version DESC
			LIMIT 1
		)
		SELECT EXTRACT(EPOCH FROM ($1::timestamptz - scheduled_at))::integer
		FROM scheduled
	`, predictedAt.UTC(), event.TripID, event.StopID, event.StopSequence).Scan(&delay)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !delay.Valid {
		return nil, nil
	}
	return delay.Int32, nil
}

func (r *PostgresRepository) CountEvents(ctx context.Context) (int64, error) {
	if r == nil || r.pool == nil {
		return 0, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*)
			 FROM vehicle_position
			 JOIN feed_snapshot ON feed_snapshot.snapshot_id = vehicle_position.snapshot_id
			 WHERE feed_snapshot.source_mode = $1) +
			(SELECT count(*)
			 FROM prediction
			 JOIN feed_snapshot ON feed_snapshot.snapshot_id = prediction.snapshot_id
			 WHERE feed_snapshot.source_mode = $1)
	`, r.sourceMode).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) LatestSnapshotAt(ctx context.Context) (time.Time, error) {
	if r == nil || r.pool == nil {
		return time.Time{}, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	var latest *time.Time
	if err := r.pool.QueryRow(ctx, `SELECT max(recorded_at) FROM feed_snapshot WHERE source_mode = $1`, r.sourceMode).Scan(&latest); err != nil {
		return time.Time{}, fmt.Errorf("load latest snapshot: %w", err)
	}
	if latest == nil {
		return time.Time{}, nil
	}
	return latest.UTC(), nil
}

func (r *PostgresRepository) LatestVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	cutoff := time.Now().UTC().Add(-r.freshnessWindow)
	freshPredictionCutoff := time.Now().UTC().Add(-15 * time.Minute)
	rows, err := r.pool.Query(ctx, `
		SELECT
			vehicle.vehicle_id,
			vehicle.route_id,
			vehicle.trip_id,
			vehicle.latitude,
			vehicle.longitude,
			vehicle.recorded_at,
			COALESCE(vehicle.delay_seconds, trip_prediction.delay_seconds) AS delay_seconds
		FROM latest_vehicle_positions AS vehicle
		JOIN feed_snapshot AS snapshot ON snapshot.snapshot_id = vehicle.snapshot_id
		LEFT JOIN LATERAL (
			SELECT prediction.delay_seconds
			FROM prediction
			JOIN feed_snapshot AS prediction_snapshot ON prediction_snapshot.snapshot_id = prediction.snapshot_id
			WHERE prediction.delay_seconds IS NOT NULL
			  AND prediction_snapshot.source_mode = $2
			  AND (
				prediction.recorded_at BETWEEN vehicle.recorded_at - INTERVAL '5 minutes'
				                           AND vehicle.recorded_at + INTERVAL '5 minutes'
				OR prediction.recorded_at >= $3
			  )
			  AND (
				prediction.trip_id = vehicle.trip_id
				OR prediction.vehicle_id = vehicle.vehicle_id
			  )
			ORDER BY
			  CASE WHEN prediction.trip_id = vehicle.trip_id THEN 0 ELSE 1 END,
			  prediction.recorded_at DESC,
			  CASE WHEN prediction.predicted_at >= CURRENT_TIMESTAMP THEN 0 ELSE 1 END,
			  CASE WHEN prediction.predicted_at >= CURRENT_TIMESTAMP THEN prediction.predicted_at END ASC NULLS LAST,
			  CASE WHEN prediction.predicted_at < CURRENT_TIMESTAMP THEN prediction.predicted_at END DESC NULLS LAST,
			  prediction.prediction_id DESC
			LIMIT 1
		) AS trip_prediction ON TRUE
		WHERE vehicle.recorded_at >= $1
		  AND snapshot.source_mode = $2
		ORDER BY vehicle.recorded_at DESC, vehicle.vehicle_id
	`, cutoff, r.sourceMode, freshPredictionCutoff)
	if err != nil {
		return nil, fmt.Errorf("query latest vehicles: %w", err)
	}
	defer rows.Close()

	vehicles := make([]domain.Vehicle, 0)
	for rows.Next() {
		var vehicle domain.Vehicle
		var delay pgtype.Int4
		if err := rows.Scan(&vehicle.VehicleID, &vehicle.RouteID, &vehicle.TripID, &vehicle.Latitude, &vehicle.Longitude, &vehicle.RecordedAt, &delay); err != nil {
			return nil, fmt.Errorf("scan latest vehicle: %w", err)
		}
		if delay.Valid {
			vehicle.DelaySeconds = int64(delay.Int32)
			vehicle.HasDelay = true
		}
		vehicles = append(vehicles, vehicle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest vehicles: %w", err)
	}
	return vehicles, nil
}

func (r *PostgresRepository) InsertArrival(ctx context.Context, arrival domain.ArrivalObserved) (bool, error) {
	if r == nil || r.pool == nil {
		return false, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	if strings.TrimSpace(arrival.TripID) == "" || strings.TrimSpace(arrival.ServiceDate) == "" || strings.TrimSpace(arrival.StopID) == "" {
		return false, fmt.Errorf("insert arrival: trip_id, service_date and stop_id are required")
	}
	if arrival.ObservedAt.IsZero() {
		return false, fmt.Errorf("insert arrival: observed_at is required")
	}
	if arrival.Method != domain.ArrivalMethodLastUpdate && arrival.Method != domain.ArrivalMethodGeofence {
		return false, fmt.Errorf("insert arrival: unsupported method %q", arrival.Method)
	}
	if arrival.Confidence < 0 || arrival.Confidence > 1 {
		return false, fmt.Errorf("insert arrival: confidence must be between 0 and 1")
	}
	if _, err := time.Parse("2006-01-02", arrival.ServiceDate); err != nil {
		return false, fmt.Errorf("insert arrival: invalid service_date: %w", err)
	}
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin arrival transaction: %w", err)
	}
	defer transaction.Rollback(ctx)
	var arrivalID int64
	err = transaction.QueryRow(ctx, `
		INSERT INTO arrival_observed
			(trip_id, service_date, stop_id, stop_sequence, observed_at, source, method, confidence, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8)
		ON CONFLICT (trip_id, service_date, stop_id, method) DO NOTHING
		RETURNING arrival_id
	`, arrival.TripID, arrival.ServiceDate, arrival.StopID, optionalSequence(arrival.StopSequence), arrival.ObservedAt.UTC(), string(arrival.Method), arrival.Confidence, arrival.Reason).Scan(&arrivalID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert arrival: %w", err)
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO prediction_error
			(prediction_id, prediction_recorded_at, arrival_id, error_seconds, within_on_time_window)
		SELECT
			prediction.prediction_id,
			prediction.recorded_at,
			$1,
			EXTRACT(EPOCH FROM ($2::timestamptz - prediction.predicted_at))::integer,
			EXTRACT(EPOCH FROM ($2::timestamptz - prediction.predicted_at)) BETWEEN -60 AND 300
		FROM prediction
		WHERE prediction.trip_id = $3
		  AND prediction.service_date = $4::date
		  AND prediction.stop_id = $5
		ON CONFLICT DO NOTHING
	`, arrivalID, arrival.ObservedAt.UTC(), arrival.TripID, arrival.ServiceDate, arrival.StopID)
	if err != nil {
		return false, fmt.Errorf("insert prediction errors: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit arrival transaction: %w", err)
	}
	return true, nil
}

func (r *PostgresRepository) LatestStops(ctx context.Context) ([]domain.Stop, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (stop.stop_id) stop.stop_id, stop.stop_lat, stop.stop_lon
		FROM gtfs_stop AS stop
		JOIN gtfs_feed_version AS version ON version.feed_version = stop.feed_version
		WHERE stop.stop_lat IS NOT NULL AND stop.stop_lon IS NOT NULL
		ORDER BY stop.stop_id, version.retrieved_at DESC, version.feed_version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query latest stops: %w", err)
	}
	defer rows.Close()
	stops := make([]domain.Stop, 0)
	for rows.Next() {
		var stop domain.Stop
		if err := rows.Scan(&stop.StopID, &stop.Latitude, &stop.Longitude); err != nil {
			return nil, fmt.Errorf("scan stop: %w", err)
		}
		stops = append(stops, stop)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stops: %w", err)
	}
	return stops, nil
}

func (r *PostgresRepository) ErrorSummary(ctx context.Context, limit int) ([]domain.ErrorSummary, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
			COALESCE(prediction.route_id, ''),
			GREATEST(0, LEAST(3600, EXTRACT(EPOCH FROM (prediction.predicted_at - prediction.recorded_at))::integer)) AS horizon_seconds,
			count(*)::bigint,
			avg(prediction_error.error_seconds)::double precision,
			avg(CASE WHEN prediction_error.within_on_time_window THEN 1.0 ELSE 0.0 END)::double precision
		FROM prediction_error
		JOIN prediction
		  ON prediction.prediction_id = prediction_error.prediction_id
		 AND prediction.recorded_at = prediction_error.prediction_recorded_at
		JOIN feed_snapshot
		  ON feed_snapshot.snapshot_id = prediction.snapshot_id
		WHERE feed_snapshot.source_mode = $2
		GROUP BY COALESCE(prediction.route_id, ''), horizon_seconds
		ORDER BY horizon_seconds, COALESCE(prediction.route_id, '')
		LIMIT $1
	`, limit, r.sourceMode)
	if err != nil {
		return nil, fmt.Errorf("query prediction error summary: %w", err)
	}
	defer rows.Close()
	result := make([]domain.ErrorSummary, 0)
	for rows.Next() {
		var summary domain.ErrorSummary
		if err := rows.Scan(&summary.RouteID, &summary.HorizonSeconds, &summary.SampleCount, &summary.MeanErrorSeconds, &summary.OnTimeRate); err != nil {
			return nil, fmt.Errorf("scan prediction error summary: %w", err)
		}
		result = append(result, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prediction error summary: %w", err)
	}
	return result, nil
}

func optionalDelay(hasDelay bool, delaySeconds int64) any {
	if !hasDelay {
		return nil
	}
	return delaySeconds
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalSequence(value uint32) any {
	if value == 0 {
		return nil
	}
	return value
}
