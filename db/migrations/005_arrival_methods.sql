ALTER TABLE arrival_observed
    ADD COLUMN IF NOT EXISTS method text,
    ADD COLUMN IF NOT EXISTS confidence double precision,
    ADD COLUMN IF NOT EXISTS reason text;

UPDATE arrival_observed
SET method = CASE
    WHEN source IN ('last_update', 'geofence') THEN source
    ELSE 'last_update'
END
WHERE method IS NULL;

UPDATE arrival_observed
SET confidence = 0.0
WHERE confidence IS NULL;

UPDATE arrival_observed
SET reason = 'legacy arrival'
WHERE reason IS NULL;

ALTER TABLE arrival_observed
    ALTER COLUMN method SET DEFAULT 'last_update',
    ALTER COLUMN method SET NOT NULL,
    ALTER COLUMN confidence SET DEFAULT 0.0,
    ALTER COLUMN confidence SET NOT NULL,
    ALTER COLUMN reason SET DEFAULT '',
    ALTER COLUMN reason SET NOT NULL;

ALTER TABLE arrival_observed
    ADD CONSTRAINT arrival_observed_method_check
    CHECK (method IN ('last_update', 'geofence'));

ALTER TABLE arrival_observed
    ADD CONSTRAINT arrival_observed_confidence_check
    CHECK (confidence >= 0 AND confidence <= 1);

CREATE UNIQUE INDEX IF NOT EXISTS arrival_observed_logical_method_idx
    ON arrival_observed (trip_id, service_date, stop_id, method);
