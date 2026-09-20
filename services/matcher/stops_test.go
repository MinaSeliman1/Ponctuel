package matcher

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadStopsURLReadsStopsFromGTFSArchive(t *testing.T) {
	archive := makeGTFSArchive(t, "stops.txt", "stop_id,stop_lat,stop_lon\nstop-1,45.5017,-73.5673\nstop-incomplete,,\n")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/zip")
		_, _ = response.Write(archive)
	}))
	defer server.Close()

	stops, err := loadStopsURL(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(stops) != 1 || stops[0].StopID != "stop-1" || stops[0].Latitude != 45.5017 || stops[0].Longitude != -73.5673 {
		t.Fatalf("loaded stops = %+v, want one complete stop", stops)
	}
}

func TestLoadStopsURLRequiresStopsFile(t *testing.T) {
	archive := makeGTFSArchive(t, "routes.txt", "route_id\n51\n")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write(archive)
	}))
	defer server.Close()

	if _, err := loadStopsURL(context.Background(), server.URL); err == nil {
		t.Fatal("loadStopsURL() error = nil, want missing stops.txt error")
	}
}

func makeGTFSArchive(t *testing.T, name, contents string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(file, contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
