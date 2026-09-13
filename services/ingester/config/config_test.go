package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultsToFixtureModeAndThirtySecondPoll(t *testing.T) {
	cfg, err := LoadFrom(testEnvironment(map[string]string{
		"DATABASE_URL": "postgres://example.invalid/ponctuel",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppEnv != "fixture" {
		t.Fatalf("APP_ENV = %q, want fixture", cfg.AppEnv)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Fatalf("poll interval = %s, want 30s", cfg.PollInterval)
	}
	if cfg.STMAPIKey != "" {
		t.Fatalf("fixture mode API key = %q, want empty", cfg.STMAPIKey)
	}
	if cfg.RedpandaGroupID == "" || cfg.GeofenceRadiusMeters <= 0 {
		t.Fatalf("matcher defaults = %#v", cfg)
	}
}

func TestLoadFromParsesRedpandaBrokers(t *testing.T) {
	cfg, err := LoadFrom(testEnvironment(map[string]string{
		"REDPANDA_BROKERS":       " redpanda:9092, localhost:19092 ",
		"GEOFENCE_RADIUS_METERS": "45",
	}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if len(cfg.RedpandaBrokers) != 2 || cfg.RedpandaBrokers[0] != "redpanda:9092" || cfg.GeofenceRadiusMeters != 45 {
		t.Fatalf("config = %#v, want parsed matcher values", cfg)
	}
}

func TestLoadSTMModeRequiresAPIKey(t *testing.T) {
	_, err := LoadFrom(testEnvironment(map[string]string{
		"APP_ENV": "stm",
	}))
	if err == nil || !strings.Contains(err.Error(), "STM_API_KEY") {
		t.Fatalf("LoadSTM without key error = %v, want missing STM_API_KEY", err)
	}
}

func TestLoadRejectsInvalidPollInterval(t *testing.T) {
	_, err := LoadFrom(testEnvironment(map[string]string{
		"POLL_INTERVAL": "not-a-duration",
	}))
	if err == nil || !strings.Contains(err.Error(), "POLL_INTERVAL") {
		t.Fatalf("invalid poll error = %v, want POLL_INTERVAL context", err)
	}
}

func testEnvironment(overrides map[string]string) func(string) (string, bool) {
	values := map[string]string{
		"APP_ENV":                   "fixture",
		"INGESTER_HTTP_ADDR":        ":8081",
		"DATABASE_URL":              "postgres://ponctuel:ponctuel@localhost:5432/ponctuel?sslmode=disable",
		"STM_TRIP_UPDATES_URL":      "https://api.stm.info/pub/od/gtfs-rt/ic/v2/tripUpdates",
		"STM_VEHICLE_POSITIONS_URL": "https://api.stm.info/pub/od/gtfs-rt/ic/v2/vehiclePositions",
		"STM_API_KEY_HEADER":        "apikey",
		"POLL_INTERVAL":             "30s",
		"FRESHNESS_WINDOW":          "2m",
		"FIXTURE_DIR":               "testdata/realtime",
		"RAW_DATA_DIR":              "./data/raw",
	}
	for key, value := range overrides {
		values[key] = value
	}
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
