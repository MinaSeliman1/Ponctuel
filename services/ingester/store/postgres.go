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
	Ping(ctx context.Context) error
}

type PostgresRepository struct {
	pool            *pgxpool.Pool
	freshnessWindow time.Duration
}

func NewPostgresRepository(pool *pgxpool.Pool, freshnessWindow time.Duration) *PostgresRepository {
	if freshnessWindow <= 0 {
		freshnessWindow = defaultFreshnessWindow
	}
	return &PostgresRepository{pool: pool, freshnessWindow: freshnessWindow}
}

func Connect(ctx context.Context, databaseURL string, freshnessWindow time.Duration) (*PostgresRepository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return NewPostgresRepository(pool, freshnessWindow), nil
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

	var sourceTimestamp any
	if !snapshot.SourceTimestamp.IsZero() {
		sourceTimestamp = snapshot.SourceTimestamp
	}
	var snapshotID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO feed_snapshot (feed_type, recorded_at, source_timestamp, payload_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (feed_type, payload_hash) DO NOTHING
		RETURNING snapshot_id
	`, string(snapshot.FeedType), snapshot.RecordedAt, sourceTimestamp, snapshot.PayloadHash).Scan(&snapshotID)
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
			_, err = transaction.Exec(ctx, `
				INSERT INTO prediction
					(snapshot_id, entity_id, recorded_at, service_date, vehicle_id, trip_id,
					 route_id, stop_id, stop_sequence, predicted_at, delay_seconds)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			`, snapshotID, event.EntityID, event.RecordedAt, predictedAt.UTC().Format("2006-01-02"), nullableString(event.VehicleID), event.TripID, nullableString(event.RouteID), event.StopID, event.StopSequence, predictedAt, delay)
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

func (r *PostgresRepository) CountEvents(ctx context.Context) (int64, error) {
	if r == nil || r.pool == nil {
		return 0, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM vehicle_position) +
		       (SELECT count(*) FROM prediction)
	`).Scan(&count)
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
	if err := r.pool.QueryRow(ctx, `SELECT max(recorded_at) FROM feed_snapshot`).Scan(&latest); err != nil {
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
	rows, err := r.pool.Query(ctx, `
		SELECT vehicle_id, route_id, trip_id, latitude, longitude, recorded_at, delay_seconds
		FROM latest_vehicle_positions
		WHERE recorded_at >= $1
		ORDER BY recorded_at DESC, vehicle_id
	`, cutoff)
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
	result, err := r.pool.Exec(ctx, `
		INSERT INTO arrival_observed
			(trip_id, service_date, stop_id, stop_sequence, observed_at, source, method, confidence, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8)
		ON CONFLICT (trip_id, service_date, stop_id, method) DO NOTHING
	`, arrival.TripID, arrival.ServiceDate, arrival.StopID, optionalSequence(arrival.StopSequence), arrival.ObservedAt.UTC(), string(arrival.Method), arrival.Confidence, arrival.Reason)
	if err != nil {
		return false, fmt.Errorf("insert arrival: %w", err)
	}
	return result.RowsAffected() == 1, nil
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
