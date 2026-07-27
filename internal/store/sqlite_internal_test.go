package store

import (
	"context"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
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

func TestSQLiteDSNNormalizesRelativePaths(t *testing.T) {
	got := sqliteDSN(filepath.Join("state", "runs.db"))
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "file" {
		t.Fatalf("sqliteDSN() scheme = %q, want file in %q", parsed.Scheme, got)
	}
	if parsed.Host != "" {
		t.Fatalf("sqliteDSN() host = %q, want empty host in %q", parsed.Host, got)
	}
	if !strings.HasSuffix(filepath.ToSlash(parsed.Path), "/state/runs.db") {
		t.Fatalf("sqliteDSN() path = %q, want absolute path ending in /state/runs.db in %q", parsed.Path, got)
	}
	pragmas := parsed.Query()["_pragma"]
	if !containsString(pragmas, "foreign_keys(1)") ||
		!containsString(pragmas, "busy_timeout(5000)") ||
		!containsString(pragmas, "journal_mode(WAL)") {
		t.Fatalf("sqliteDSN() pragmas = %v, want foreign_keys, busy_timeout, and journal_mode", pragmas)
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
