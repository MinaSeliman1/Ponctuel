package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAppEnv           = "fixture"
	defaultHTTPAddr         = ":8081"
	defaultDatabaseURL      = "postgres://ponctuel:ponctuel@localhost:5432/ponctuel?sslmode=disable"
	defaultTripUpdatesURL   = "https://api.stm.info/pub/od/gtfs-rt/ic/v2/tripUpdates"
	defaultVehicleURL       = "https://api.stm.info/pub/od/gtfs-rt/ic/v2/vehiclePositions"
	defaultAPIKeyHeader     = "apikey"
	defaultPollInterval     = 30 * time.Second
	defaultFreshnessWindow  = 2 * time.Minute
	defaultFixtureDirectory = "testdata/realtime"
	defaultRawDataDirectory = "./data/raw"
	defaultRedpandaGroupID  = "ponctuel-matcher"
	defaultGeofenceRadius   = 60.0
)

type Config struct {
	AppEnv               string
	HTTPAddr             string
	DatabaseURL          string
	PollInterval         time.Duration
	FreshnessWindow      time.Duration
	FixtureDir           string
	RawDataDir           string
	STMTripUpdatesURL    string
	STMVehicleURL        string
	STMAPIKey            string
	STMAPIKeyHeader      string
	RedpandaBrokers      []string
	RedpandaGroupID      string
	GeofenceRadiusMeters float64
}

func Load() (Config, error) {
	return LoadFrom(os.LookupEnv)
}

func LoadFrom(lookup func(string) (string, bool)) (Config, error) {
	if lookup == nil {
		return Config{}, fmt.Errorf("configuration environment lookup is nil")
	}

	cfg := Config{
		AppEnv:            strings.ToLower(valueOr(lookup, "APP_ENV", defaultAppEnv)),
		HTTPAddr:          valueOr(lookup, "INGESTER_HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL:       valueOr(lookup, "DATABASE_URL", defaultDatabaseURL),
		FixtureDir:        valueOr(lookup, "FIXTURE_DIR", defaultFixtureDirectory),
		RawDataDir:        valueOr(lookup, "RAW_DATA_DIR", defaultRawDataDirectory),
		STMTripUpdatesURL: valueOr(lookup, "STM_TRIP_UPDATES_URL", defaultTripUpdatesURL),
		STMVehicleURL:     valueOr(lookup, "STM_VEHICLE_POSITIONS_URL", defaultVehicleURL),
		STMAPIKey:         valueOr(lookup, "STM_API_KEY", ""),
		STMAPIKeyHeader:   valueOr(lookup, "STM_API_KEY_HEADER", defaultAPIKeyHeader),
		RedpandaBrokers:   csvValues(valueOr(lookup, "REDPANDA_BROKERS", "")),
		RedpandaGroupID:   valueOr(lookup, "REDPANDA_GROUP_ID", defaultRedpandaGroupID),
	}

	var err error
	if cfg.PollInterval, err = durationValue(lookup, "POLL_INTERVAL", defaultPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.FreshnessWindow, err = durationValue(lookup, "FRESHNESS_WINDOW", defaultFreshnessWindow); err != nil {
		return Config{}, err
	}
	if cfg.GeofenceRadiusMeters, err = floatValue(lookup, "GEOFENCE_RADIUS_METERS", defaultGeofenceRadius); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.AppEnv)) {
	case "fixture", "stm":
	default:
		return fmt.Errorf("APP_ENV must be fixture or stm")
	}
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return fmt.Errorf("INGESTER_HTTP_ADDR is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if err := validateDatabaseURL(c.DatabaseURL); err != nil {
		return err
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("POLL_INTERVAL must be positive")
	}
	if c.FreshnessWindow <= 0 {
		return fmt.Errorf("FRESHNESS_WINDOW must be positive")
	}
	if c.GeofenceRadiusMeters != 0 && (c.GeofenceRadiusMeters < 0 || c.GeofenceRadiusMeters > 1000) {
		return fmt.Errorf("GEOFENCE_RADIUS_METERS must be between 0 and 1000")
	}
	if strings.TrimSpace(c.FixtureDir) == "" {
		return fmt.Errorf("FIXTURE_DIR is required")
	}
	if strings.TrimSpace(c.STMAPIKeyHeader) == "" || strings.ContainsAny(c.STMAPIKeyHeader, " \t\r\n") {
		return fmt.Errorf("STM_API_KEY_HEADER must be a non-empty HTTP header name")
	}
	if err := validateURL("STM_TRIP_UPDATES_URL", c.STMTripUpdatesURL); err != nil {
		return err
	}
	if err := validateURL("STM_VEHICLE_POSITIONS_URL", c.STMVehicleURL); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(c.AppEnv), "stm") && strings.TrimSpace(c.STMAPIKey) == "" {
		return fmt.Errorf("STM_API_KEY is required when APP_ENV=stm")
	}
	return nil
}

func valueOr(lookup func(string) (string, bool), key, fallback string) string {
	if value, ok := lookup(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func durationValue(lookup func(string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value := valueOr(lookup, key, fallback.String())
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}

func floatValue(lookup func(string) (string, bool), key string, fallback float64) (float64, error) {
	value := valueOr(lookup, key, strconv.FormatFloat(fallback, 'f', -1, 64))
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", key)
	}
	return parsed, nil
}

func csvValues(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func validateURL(name, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL", name)
	}
	return nil
}

func validateDatabaseURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return fmt.Errorf("DATABASE_URL must be a PostgreSQL URL")
	}
	return nil
}
