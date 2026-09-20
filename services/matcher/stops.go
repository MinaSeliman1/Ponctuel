package matcher

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"ponctuel/services/ingester/domain"
)

const maxGTFSArchiveBytes = 64 * 1024 * 1024

func loadStopsURL(ctx context.Context, rawURL string) ([]domain.Stop, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("GTFS static URL is required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build GTFS static request: %w", err)
	}
	request.Header.Set("Accept", "application/zip")
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("request GTFS static feed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request GTFS static feed: status %d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxGTFSArchiveBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read GTFS static feed: %w", err)
	}
	if int64(len(payload)) > maxGTFSArchiveBytes {
		return nil, fmt.Errorf("GTFS static feed exceeds %d bytes", maxGTFSArchiveBytes)
	}
	archive, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, fmt.Errorf("open GTFS static archive: %w", err)
	}
	for _, archiveFile := range archive.File {
		if !strings.EqualFold(path.Base(archiveFile.Name), "stops.txt") {
			continue
		}
		return readStopsCSV(archiveFile)
	}
	return nil, fmt.Errorf("GTFS static archive does not contain stops.txt")
}

func readStopsCSV(archiveFile *zip.File) ([]domain.Stop, error) {
	reader, err := archiveFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open stops.txt: %w", err)
	}
	defer reader.Close()

	csvReader := csv.NewReader(io.LimitReader(reader, maxGTFSArchiveBytes))
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read stops.txt header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for index, column := range header {
		columns[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(column, "\ufeff")))] = index
	}
	for _, required := range []string{"stop_id", "stop_lat", "stop_lon"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("stops.txt is missing %s", required)
		}
	}

	stops := make([]domain.Stop, 0)
	for {
		record, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read stops.txt row: %w", readErr)
		}
		if len(record) <= columns["stop_lon"] || len(record) <= columns["stop_lat"] {
			continue
		}
		stopID := strings.TrimSpace(record[columns["stop_id"]])
		latitude, latitudeErr := strconv.ParseFloat(strings.TrimSpace(record[columns["stop_lat"]]), 64)
		longitude, longitudeErr := strconv.ParseFloat(strings.TrimSpace(record[columns["stop_lon"]]), 64)
		if stopID == "" || latitudeErr != nil || longitudeErr != nil || !validStopCoordinate(latitude, longitude) {
			continue
		}
		stops = append(stops, domain.Stop{StopID: stopID, Latitude: latitude, Longitude: longitude})
	}
	if len(stops) == 0 {
		return nil, fmt.Errorf("stops.txt contains no complete stop coordinates")
	}
	return stops, nil
}

func validStopCoordinate(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}
