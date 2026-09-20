package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ponctuel/services/ingester/gtfs"
)

const (
	defaultStaticGTFSURL = "https://www.stm.info/sites/default/files/gtfs/gtfs_stm.zip"
	maxStaticGTFSBytes   = 64 << 20
)

// EnsureStaticSchedule imports the current STM schedule only when the database
// has no static feed yet. Realtime predictions can then derive delay values
// from predicted time minus scheduled stop time when the feed omits delay.
func (r *PostgresRepository) EnsureStaticSchedule(ctx context.Context, sourceURL string) (bool, error) {
	if r == nil || r.pool == nil {
		return false, fmt.Errorf("PostgreSQL repository is not initialized")
	}
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM gtfs_feed_version)`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check static GTFS schedule: %w", err)
	}
	if exists {
		return false, nil
	}

	if strings.TrimSpace(sourceURL) == "" {
		sourceURL = defaultStaticGTFSURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return false, fmt.Errorf("build static GTFS request: %w", err)
	}
	request.Header.Set("Accept", "application/zip, application/octet-stream")
	response, err := (&http.Client{Timeout: 45 * time.Second}).Do(request)
	if err != nil {
		return false, fmt.Errorf("download static GTFS schedule: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return false, fmt.Errorf("download static GTFS schedule: HTTP %d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxStaticGTFSBytes+1))
	if err != nil {
		return false, fmt.Errorf("read static GTFS schedule: %w", err)
	}
	if int64(len(payload)) > maxStaticGTFSBytes {
		return false, fmt.Errorf("static GTFS schedule exceeds %d bytes", maxStaticGTFSBytes)
	}
	feedVersion := "stm-" + time.Now().UTC().Format("2006-01-02")
	if err := gtfs.Import(ctx, r.pool, bytes.NewReader(payload), feedVersion); err != nil {
		return false, fmt.Errorf("import static GTFS schedule: %w", err)
	}
	return true, nil
}
