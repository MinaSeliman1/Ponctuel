package store

import (
	"context"
	"errors"
	"fmt"
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
