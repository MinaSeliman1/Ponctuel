CREATE TABLE IF NOT EXISTS feed_snapshot (
    snapshot_id bigserial PRIMARY KEY,
    feed_type text NOT NULL CHECK (feed_type IN ('trip_updates', 'vehicle_positions')),
    recorded_at timestamptz NOT NULL,
    source_timestamp timestamptz,
    payload_hash text NOT NULL CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (feed_type, payload_hash)
);

CREATE INDEX IF NOT EXISTS feed_snapshot_recorded_at_idx
    ON feed_snapshot (recorded_at DESC);

CREATE TABLE IF NOT EXISTS vehicle_position (
    event_id bigserial NOT NULL,
    snapshot_id bigint NOT NULL REFERENCES feed_snapshot(snapshot_id) ON DELETE RESTRICT,
    entity_id text NOT NULL,
    recorded_at timestamptz NOT NULL,
    vehicle_id text NOT NULL,
    trip_id text,
    route_id text,
    latitude double precision NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude double precision NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    delay_seconds integer,
    PRIMARY KEY (event_id, recorded_at)
);

CREATE INDEX IF NOT EXISTS vehicle_position_route_recorded_idx
    ON vehicle_position (route_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS vehicle_position_vehicle_recorded_idx
    ON vehicle_position (vehicle_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS prediction (
    prediction_id bigserial NOT NULL,
    snapshot_id bigint NOT NULL REFERENCES feed_snapshot(snapshot_id) ON DELETE RESTRICT,
    entity_id text NOT NULL,
    recorded_at timestamptz NOT NULL,
    service_date date NOT NULL,
    vehicle_id text,
    trip_id text NOT NULL,
    route_id text,
    stop_id text NOT NULL,
    stop_sequence integer,
    predicted_at timestamptz NOT NULL,
    delay_seconds integer,
    PRIMARY KEY (prediction_id, recorded_at)
);

CREATE INDEX IF NOT EXISTS prediction_trip_stop_recorded_idx
    ON prediction (trip_id, service_date, stop_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS prediction_route_recorded_idx
    ON prediction (route_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS arrival_observed (
    arrival_id bigserial PRIMARY KEY,
    trip_id text NOT NULL,
    service_date date NOT NULL,
    stop_id text NOT NULL,
    stop_sequence integer,
    observed_at timestamptz NOT NULL,
    source text NOT NULL,
    UNIQUE (trip_id, service_date, stop_id, observed_at)
);

CREATE INDEX IF NOT EXISTS arrival_observed_trip_stop_idx
    ON arrival_observed (trip_id, service_date, stop_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS prediction_error (
    prediction_id bigint NOT NULL,
    prediction_recorded_at timestamptz NOT NULL,
    arrival_id bigint NOT NULL REFERENCES arrival_observed(arrival_id) ON DELETE CASCADE,
    error_seconds integer NOT NULL,
    within_on_time_window boolean NOT NULL,
    calculated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (prediction_id, prediction_recorded_at, arrival_id),
    FOREIGN KEY (prediction_id, prediction_recorded_at)
        REFERENCES prediction(prediction_id, recorded_at) ON DELETE CASCADE
);

-- Hypertables are an optional local optimization. Supabase PostgreSQL does
-- not provide TimescaleDB, so the same schema remains portable there.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_extension
        WHERE extname = 'timescaledb'
    ) THEN
        EXECUTE 'SELECT create_hypertable(''vehicle_position'', by_range(''recorded_at''), if_not_exists => TRUE)';
        EXECUTE 'SELECT create_hypertable(''prediction'', by_range(''recorded_at''), if_not_exists => TRUE)';
    END IF;
END
$$;
