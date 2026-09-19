-- Supabase Free exposes standard PostgreSQL without TimescaleDB. The local
-- TimescaleDB image still enables the extension when it is available.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_available_extensions
        WHERE name = 'timescaledb'
    ) THEN
        EXECUTE 'CREATE EXTENSION IF NOT EXISTS timescaledb';
    END IF;
END
$$;
