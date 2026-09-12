package gtfs

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxArchiveBytes int64 = 256 << 20

const staticSourceURL = "https://www.stm.info/sites/default/files/gtfs/gtfs_stm.zip"

var requiredFiles = []string{"agency.txt", "routes.txt", "trips.txt", "stops.txt", "stop_times.txt"}

// ValidateRequiredFiles checks the minimum static GTFS surface used by Ponctuel.
func ValidateRequiredFiles(zipReader *zip.Reader) error {
	if zipReader == nil {
		return fmt.Errorf("GTFS archive is nil")
	}
	for _, required := range requiredFiles {
		if _, err := openZipFile(zipReader, required); err != nil {
			return err
		}
	}
	return nil
}

// Import validates and imports one versioned static GTFS archive transactionally.
// A feed version is immutable: importing it twice is rejected.
func Import(ctx context.Context, db *pgxpool.Pool, archive io.Reader, feedVersion string) error {
	if ctx == nil {
		return fmt.Errorf("GTFS import context is nil")
	}
	if archive == nil {
		return fmt.Errorf("GTFS archive is nil")
	}
	feedVersion = strings.TrimSpace(feedVersion)
	if feedVersion == "" {
		return fmt.Errorf("feed version is required")
	}

	payload, err := io.ReadAll(io.LimitReader(archive, maxArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("read GTFS archive: %w", err)
	}
	if int64(len(payload)) > maxArchiveBytes {
		return fmt.Errorf("GTFS archive exceeds %d bytes", maxArchiveBytes)
	}
	zipReader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return fmt.Errorf("open GTFS archive: %w", err)
	}
	if err := ValidateRequiredFiles(zipReader); err != nil {
		return err
	}
	data, err := parseArchive(zipReader)
	if err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("GTFS import database is nil")
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin GTFS import: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
		INSERT INTO gtfs_feed_version (feed_version, source_url)
		VALUES ($1, $2)
		ON CONFLICT (feed_version) DO NOTHING
	`, feedVersion, staticSourceURL)
	if err != nil {
		return fmt.Errorf("insert feed version: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("feed version %q already exists", feedVersion)
	}
	if err := createStagingTables(ctx, tx); err != nil {
		return err
	}
	if err := stageRows(ctx, tx, feedVersion, data); err != nil {
		return err
	}
	if err := copyStagedRows(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit GTFS import: %w", err)
	}
	return nil
}

type staticData struct {
	agencies  []agencyRow
	routes    []routeRow
	trips     []tripRow
	stops     []stopRow
	stopTimes []stopTimeRow
}

type agencyRow struct {
	id, name, url, timezone string
}

type routeRow struct {
	id, agencyID, shortName, longName string
	routeType                         *int
}

type tripRow struct {
	id, routeID, serviceID, headsign string
	directionID                      *int
}

type stopRow struct {
	id, name            string
	latitude, longitude *float64
}

type stopTimeRow struct {
	tripID, stopID     string
	arrival, departure *int
	sequence           int
}

func parseArchive(zipReader *zip.Reader) (staticData, error) {
	var data staticData
	var err error
	if data.agencies, err = parseAgencies(zipReader); err != nil {
		return staticData{}, err
	}
	if data.routes, err = parseRoutes(zipReader); err != nil {
		return staticData{}, err
	}
	if data.trips, err = parseTrips(zipReader); err != nil {
		return staticData{}, err
	}
	if data.stops, err = parseStops(zipReader); err != nil {
		return staticData{}, err
	}
	if data.stopTimes, err = parseStopTimes(zipReader); err != nil {
		return staticData{}, err
	}
	return data, nil
}

type csvFile struct {
	name    string
	columns map[string]int
	rows    [][]string
}

func readCSV(zipReader *zip.Reader, name string, required ...string) (csvFile, error) {
	file, err := openZipFile(zipReader, name)
	if err != nil {
		return csvFile{}, err
	}
	reader, err := file.Open()
	if err != nil {
		return csvFile{}, fmt.Errorf("open %s: %w", name, err)
	}
	defer reader.Close()

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return csvFile{}, fmt.Errorf("%s header: %w", name, err)
	}
	columns := make(map[string]int, len(header))
	for index, column := range header {
		column = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(column, "\ufeff")))
		if column == "" {
			return csvFile{}, fmt.Errorf("%s header column %d is empty", name, index+1)
		}
		if _, exists := columns[column]; exists {
			return csvFile{}, fmt.Errorf("%s header contains duplicate column %q", name, column)
		}
		columns[column] = index
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			return csvFile{}, fmt.Errorf("%s: missing required column %q", name, column)
		}
	}
	rows := make([][]string, 0)
	rowNumber := 1
	for {
		rowNumber++
		row, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return csvFile{}, fmt.Errorf("%s row %d: %w", name, rowNumber, readErr)
		}
		if blankRow(row) {
			continue
		}
		if len(row) != len(header) {
			return csvFile{}, fmt.Errorf("%s row %d: got %d fields, want %d", name, rowNumber, len(row), len(header))
		}
		rows = append(rows, row)
	}
	return csvFile{name: name, columns: columns, rows: rows}, nil
}

func parseAgencies(zipReader *zip.Reader) ([]agencyRow, error) {
	file, err := readCSV(zipReader, "agency.txt", "agency_id", "agency_name")
	if err != nil {
		return nil, err
	}
	rows := make([]agencyRow, 0, len(file.rows))
	for index, row := range file.rows {
		agencyID, err := requiredValue(file, row, "agency_id", index)
		if err != nil {
			return nil, err
		}
		name, err := requiredValue(file, row, "agency_name", index)
		if err != nil {
			return nil, err
		}
		rows = append(rows, agencyRow{id: agencyID, name: name, url: value(file, row, "agency_url"), timezone: value(file, row, "agency_timezone")})
	}
	return rows, nil
}

func parseRoutes(zipReader *zip.Reader) ([]routeRow, error) {
	file, err := readCSV(zipReader, "routes.txt", "route_id")
	if err != nil {
		return nil, err
	}
	rows := make([]routeRow, 0, len(file.rows))
	for index, row := range file.rows {
		routeID, err := requiredValue(file, row, "route_id", index)
		if err != nil {
			return nil, err
		}
		routeType, err := optionalInt(file, row, "route_type", index)
		if err != nil {
			return nil, err
		}
		rows = append(rows, routeRow{id: routeID, agencyID: value(file, row, "agency_id"), shortName: value(file, row, "route_short_name"), longName: value(file, row, "route_long_name"), routeType: routeType})
	}
	return rows, nil
}

func parseTrips(zipReader *zip.Reader) ([]tripRow, error) {
	file, err := readCSV(zipReader, "trips.txt", "trip_id")
	if err != nil {
		return nil, err
	}
	rows := make([]tripRow, 0, len(file.rows))
	for index, row := range file.rows {
		tripID, err := requiredValue(file, row, "trip_id", index)
		if err != nil {
			return nil, err
		}
		directionID, err := optionalInt(file, row, "direction_id", index)
		if err != nil {
			return nil, err
		}
		rows = append(rows, tripRow{id: tripID, routeID: value(file, row, "route_id"), serviceID: value(file, row, "service_id"), headsign: value(file, row, "trip_headsign"), directionID: directionID})
	}
	return rows, nil
}

func parseStops(zipReader *zip.Reader) ([]stopRow, error) {
	file, err := readCSV(zipReader, "stops.txt", "stop_id", "stop_name")
	if err != nil {
		return nil, err
	}
	rows := make([]stopRow, 0, len(file.rows))
	for index, row := range file.rows {
		stopID, err := requiredValue(file, row, "stop_id", index)
		if err != nil {
			return nil, err
		}
		name, err := requiredValue(file, row, "stop_name", index)
		if err != nil {
			return nil, err
		}
		latitude, err := optionalFloat(file, row, "stop_lat", index)
		if err != nil {
			return nil, err
		}
		longitude, err := optionalFloat(file, row, "stop_lon", index)
		if err != nil {
			return nil, err
		}
		if latitude != nil && (*latitude < -90 || *latitude > 90 || math.IsNaN(*latitude)) {
			return nil, rowError(file, index, "stop_lat", "coordinate is outside [-90, 90]")
		}
		if longitude != nil && (*longitude < -180 || *longitude > 180 || math.IsNaN(*longitude)) {
			return nil, rowError(file, index, "stop_lon", "coordinate is outside [-180, 180]")
		}
		rows = append(rows, stopRow{id: stopID, name: name, latitude: latitude, longitude: longitude})
	}
	return rows, nil
}

func parseStopTimes(zipReader *zip.Reader) ([]stopTimeRow, error) {
	file, err := readCSV(zipReader, "stop_times.txt", "trip_id", "stop_id", "stop_sequence", "arrival_time", "departure_time")
	if err != nil {
		return nil, err
	}
	rows := make([]stopTimeRow, 0, len(file.rows))
	for index, row := range file.rows {
		tripID, err := requiredValue(file, row, "trip_id", index)
		if err != nil {
			return nil, err
		}
		stopID, err := requiredValue(file, row, "stop_id", index)
		if err != nil {
			return nil, err
		}
		sequenceText, err := requiredValue(file, row, "stop_sequence", index)
		if err != nil {
			return nil, err
		}
		sequence, parseErr := strconv.Atoi(sequenceText)
		if parseErr != nil || sequence < 0 {
			return nil, rowError(file, index, "stop_sequence", "must be a non-negative integer")
		}
		arrival, err := optionalGTFSSeconds(file, row, "arrival_time", index)
		if err != nil {
			return nil, err
		}
		departure, err := optionalGTFSSeconds(file, row, "departure_time", index)
		if err != nil {
			return nil, err
		}
		rows = append(rows, stopTimeRow{tripID: tripID, stopID: stopID, sequence: sequence, arrival: arrival, departure: departure})
	}
	return rows, nil
}

func createStagingTables(ctx context.Context, tx pgx.Tx) error {
	statements := []string{
		`CREATE TEMP TABLE import_agency ON COMMIT DROP AS SELECT feed_version, agency_id, agency_name, agency_url, agency_timezone FROM gtfs_agency WITH NO DATA`,
		`CREATE TEMP TABLE import_route ON COMMIT DROP AS SELECT feed_version, route_id, agency_id, route_short_name, route_long_name, route_type FROM gtfs_route WITH NO DATA`,
		`CREATE TEMP TABLE import_trip ON COMMIT DROP AS SELECT feed_version, trip_id, route_id, service_id, trip_headsign, direction_id FROM gtfs_trip WITH NO DATA`,
		`CREATE TEMP TABLE import_stop ON COMMIT DROP AS SELECT feed_version, stop_id, stop_name, stop_lat, stop_lon FROM gtfs_stop WITH NO DATA`,
		`CREATE TEMP TABLE import_stop_time ON COMMIT DROP AS SELECT feed_version, trip_id, stop_sequence, stop_id, arrival_seconds, departure_seconds FROM gtfs_stop_time WITH NO DATA`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("create GTFS staging table: %w", err)
		}
	}
	return nil
}

func stageRows(ctx context.Context, tx pgx.Tx, feedVersion string, data staticData) error {
	for _, row := range data.agencies {
		if _, err := tx.Exec(ctx, `INSERT INTO import_agency (feed_version, agency_id, agency_name, agency_url, agency_timezone) VALUES ($1, $2, $3, $4, $5)`, feedVersion, row.id, row.name, nullableString(row.url), nullableString(row.timezone)); err != nil {
			return fmt.Errorf("stage agency %q: %w", row.id, err)
		}
	}
	for _, row := range data.routes {
		if _, err := tx.Exec(ctx, `INSERT INTO import_route (feed_version, route_id, agency_id, route_short_name, route_long_name, route_type) VALUES ($1, $2, $3, $4, $5, $6)`, feedVersion, row.id, nullableString(row.agencyID), nullableString(row.shortName), nullableString(row.longName), optionalIntValue(row.routeType)); err != nil {
			return fmt.Errorf("stage route %q: %w", row.id, err)
		}
	}
	for _, row := range data.trips {
		if _, err := tx.Exec(ctx, `INSERT INTO import_trip (feed_version, trip_id, route_id, service_id, trip_headsign, direction_id) VALUES ($1, $2, $3, $4, $5, $6)`, feedVersion, row.id, nullableString(row.routeID), nullableString(row.serviceID), nullableString(row.headsign), optionalIntValue(row.directionID)); err != nil {
			return fmt.Errorf("stage trip %q: %w", row.id, err)
		}
	}
	for _, row := range data.stops {
		if _, err := tx.Exec(ctx, `INSERT INTO import_stop (feed_version, stop_id, stop_name, stop_lat, stop_lon) VALUES ($1, $2, $3, $4, $5)`, feedVersion, row.id, row.name, optionalFloatValue(row.latitude), optionalFloatValue(row.longitude)); err != nil {
			return fmt.Errorf("stage stop %q: %w", row.id, err)
		}
	}
	for _, row := range data.stopTimes {
		if _, err := tx.Exec(ctx, `INSERT INTO import_stop_time (feed_version, trip_id, stop_sequence, stop_id, arrival_seconds, departure_seconds) VALUES ($1, $2, $3, $4, $5, $6)`, feedVersion, row.tripID, row.sequence, row.stopID, optionalIntValue(row.arrival), optionalIntValue(row.departure)); err != nil {
			return fmt.Errorf("stage stop time %s/%s/%d: %w", row.tripID, row.stopID, row.sequence, err)
		}
	}
	return nil
}

func copyStagedRows(ctx context.Context, tx pgx.Tx) error {
	statements := []string{
		`INSERT INTO gtfs_agency (feed_version, agency_id, agency_name, agency_url, agency_timezone) SELECT feed_version, agency_id, agency_name, agency_url, agency_timezone FROM import_agency`,
		`INSERT INTO gtfs_route (feed_version, route_id, agency_id, route_short_name, route_long_name, route_type) SELECT feed_version, route_id, agency_id, route_short_name, route_long_name, route_type FROM import_route`,
		`INSERT INTO gtfs_trip (feed_version, trip_id, route_id, service_id, trip_headsign, direction_id) SELECT feed_version, trip_id, route_id, service_id, trip_headsign, direction_id FROM import_trip`,
		`INSERT INTO gtfs_stop (feed_version, stop_id, stop_name, stop_lat, stop_lon) SELECT feed_version, stop_id, stop_name, stop_lat, stop_lon FROM import_stop`,
		`INSERT INTO gtfs_stop_time (feed_version, trip_id, stop_sequence, stop_id, arrival_seconds, departure_seconds) SELECT feed_version, trip_id, stop_sequence, stop_id, arrival_seconds, departure_seconds FROM import_stop_time`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("copy GTFS staging rows: %w", err)
		}
	}
	return nil
}

func openZipFile(zipReader *zip.Reader, wanted string) (*zip.File, error) {
	for _, file := range zipReader.File {
		if !file.FileInfo().IsDir() && strings.EqualFold(filepath.Base(file.Name), wanted) {
			return file, nil
		}
	}
	return nil, fmt.Errorf("GTFS archive is missing %s", wanted)
}

func blankRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func value(file csvFile, row []string, column string) string {
	index, ok := file.columns[column]
	if !ok || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func requiredValue(file csvFile, row []string, column string, rowIndex int) (string, error) {
	value := value(file, row, column)
	if value == "" {
		return "", rowError(file, rowIndex, column, "value is required")
	}
	return value, nil
}

func rowError(file csvFile, rowIndex int, column, message string) error {
	return fmt.Errorf("%s row %d column %s: %s", file.name, rowIndex+2, column, message)
}

func optionalInt(file csvFile, row []string, column string, rowIndex int) (*int, error) {
	text := value(file, row, column)
	if text == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return nil, rowError(file, rowIndex, column, "must be an integer")
	}
	return &parsed, nil
}

func optionalFloat(file csvFile, row []string, column string, rowIndex int) (*float64, error) {
	text := value(file, row, column)
	if text == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsInf(parsed, 0) {
		return nil, rowError(file, rowIndex, column, "must be a finite number")
	}
	return &parsed, nil
}

func optionalGTFSSeconds(file csvFile, row []string, column string, rowIndex int) (*int, error) {
	text := value(file, row, column)
	if text == "" {
		return nil, nil
	}
	parts := strings.Split(text, ":")
	if len(parts) != 3 {
		return nil, rowError(file, rowIndex, column, "must use HH:MM:SS")
	}
	hours, hourErr := strconv.ParseInt(parts[0], 10, 64)
	minutes, minuteErr := strconv.ParseInt(parts[1], 10, 64)
	seconds, secondErr := strconv.ParseInt(parts[2], 10, 64)
	if hourErr != nil || minuteErr != nil || secondErr != nil || hours < 0 || minutes < 0 || minutes > 59 || seconds < 0 || seconds > 59 {
		return nil, rowError(file, rowIndex, column, "must use a valid GTFS time")
	}
	total := hours*3600 + minutes*60 + seconds
	if total > int64(^uint(0)>>1) {
		return nil, rowError(file, rowIndex, column, "time is too large")
	}
	parsed := int(total)
	return &parsed, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalIntValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalFloatValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
