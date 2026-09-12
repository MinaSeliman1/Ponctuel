CREATE TABLE IF NOT EXISTS gtfs_feed_version (
    feed_version text PRIMARY KEY,
    source_url text NOT NULL,
    retrieved_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS gtfs_agency (
    feed_version text NOT NULL REFERENCES gtfs_feed_version(feed_version) ON DELETE CASCADE,
    agency_id text NOT NULL,
    agency_name text NOT NULL,
    agency_url text,
    agency_timezone text,
    PRIMARY KEY (feed_version, agency_id)
);

CREATE TABLE IF NOT EXISTS gtfs_route (
    feed_version text NOT NULL REFERENCES gtfs_feed_version(feed_version) ON DELETE CASCADE,
    route_id text NOT NULL,
    agency_id text,
    route_short_name text,
    route_long_name text,
    route_type integer,
    PRIMARY KEY (feed_version, route_id)
);

CREATE TABLE IF NOT EXISTS gtfs_trip (
    feed_version text NOT NULL REFERENCES gtfs_feed_version(feed_version) ON DELETE CASCADE,
    trip_id text NOT NULL,
    route_id text,
    service_id text,
    trip_headsign text,
    direction_id integer,
    PRIMARY KEY (feed_version, trip_id)
);

CREATE TABLE IF NOT EXISTS gtfs_stop (
    feed_version text NOT NULL REFERENCES gtfs_feed_version(feed_version) ON DELETE CASCADE,
    stop_id text NOT NULL,
    stop_name text NOT NULL,
    stop_lat double precision,
    stop_lon double precision,
    PRIMARY KEY (feed_version, stop_id)
);

CREATE TABLE IF NOT EXISTS gtfs_stop_time (
    feed_version text NOT NULL REFERENCES gtfs_feed_version(feed_version) ON DELETE CASCADE,
    trip_id text NOT NULL,
    stop_sequence integer NOT NULL,
    stop_id text NOT NULL,
    arrival_seconds integer,
    departure_seconds integer,
    PRIMARY KEY (feed_version, trip_id, stop_sequence),
    FOREIGN KEY (feed_version, trip_id) REFERENCES gtfs_trip(feed_version, trip_id) ON DELETE CASCADE,
    FOREIGN KEY (feed_version, stop_id) REFERENCES gtfs_stop(feed_version, stop_id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS gtfs_stop_time_stop_lookup
    ON gtfs_stop_time (feed_version, stop_id, trip_id, stop_sequence);
