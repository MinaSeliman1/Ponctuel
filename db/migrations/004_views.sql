CREATE OR REPLACE VIEW latest_vehicle_positions AS
SELECT DISTINCT ON (vehicle_id)
    event_id,
    vehicle_id,
    route_id,
    trip_id,
    latitude,
    longitude,
    recorded_at,
    delay_seconds
FROM vehicle_position
ORDER BY vehicle_id, recorded_at DESC, event_id DESC;

CREATE OR REPLACE VIEW public_event_count AS
SELECT
    (SELECT count(*) FROM vehicle_position) +
    (SELECT count(*) FROM prediction) AS event_count;
