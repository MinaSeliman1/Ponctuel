package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOrderedMigrationFilesReturnsOnlySortedSQLFiles(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"003_realtime.sql", "README.md", "001_extensions.sql", "002_static_gtfs.sql"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("-- test\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := orderedMigrationFiles(directory)
	if err != nil {
		t.Fatalf("orderedMigrationFiles() error = %v", err)
	}
	want := []string{
		filepath.Join(directory, "001_extensions.sql"),
		filepath.Join(directory, "002_static_gtfs.sql"),
		filepath.Join(directory, "003_realtime.sql"),
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("orderedMigrationFiles() = %#v, want %#v", files, want)
	}
}
