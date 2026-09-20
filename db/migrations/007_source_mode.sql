ALTER TABLE feed_snapshot
    ADD COLUMN IF NOT EXISTS source_mode text NOT NULL DEFAULT 'fixture';

ALTER TABLE feed_snapshot
    DROP CONSTRAINT IF EXISTS feed_snapshot_source_mode_check;

ALTER TABLE feed_snapshot
    ADD CONSTRAINT feed_snapshot_source_mode_check
    CHECK (source_mode IN ('fixture', 'stm'));

CREATE INDEX IF NOT EXISTS feed_snapshot_mode_recorded_at_idx
    ON feed_snapshot (source_mode, recorded_at DESC);
