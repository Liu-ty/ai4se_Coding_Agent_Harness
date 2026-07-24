package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func TestSQLiteConnectionPragmasSurviveConnectionReplacement(t *testing.T) {
	s, err := OpenSQLite(filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.db.SetMaxIdleConns(0)

	ctx := context.Background()
	var foreignKeys, busyTimeout int
	if err := s.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 || busyTimeout != 5000 {
		t.Fatalf("replacement connection pragmas = foreign_keys:%d busy_timeout:%d, want 1 and 5000", foreignKeys, busyTimeout)
	}
}

func TestHashEventKnownAnswer(t *testing.T) {
	event := domain.RunEvent{
		RunID:        "run-42",
		Sequence:     7,
		Type:         "Recorded",
		At:           time.Unix(0, 1_234_567_890).UTC(),
		Payload:      json.RawMessage(`{"ok":true}`),
		PreviousHash: "abc123",
	}
	const want = "0e3fcb828f1f22efa11e11b9f9ef51f7cfdd8948a871bb5608761f7c42706c9f"
	if got := hashEvent(event); got != want {
		t.Fatalf("hashEvent() = %q, want %q", got, want)
	}
}
