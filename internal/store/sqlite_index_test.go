package store

import (
	"path/filepath"
	"testing"
)

func TestSQLiteCreatesRecentRunsOrderingIndex(t *testing.T) {
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	var sql string
	if err := database.db.QueryRow(
		"SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'runs_updated_id_idx'",
	).Scan(&sql); err != nil {
		t.Fatalf("recent-runs index missing: %v", err)
	}
	if sql == "" {
		t.Fatal("recent-runs index has no schema")
	}
}
