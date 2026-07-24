package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initialMigration string

// SQLiteStore persists store records in a local SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

const zeroTimeMarker int64 = -1 << 63

// OpenSQLite opens a SQLite store and applies the initial schema.
func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("read journal mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: journal mode is %q", journalMode)
	}
	if _, err := db.Exec(initialMigration); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply initial migration: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func sqliteDSN(path string) string {
	dsnPath := filepath.ToSlash(path)
	if volume := filepath.VolumeName(path); volume != "" && !strings.HasPrefix(dsnPath, "/") {
		dsnPath = "/" + dsnPath
	}
	pragmas := url.Values{}
	pragmas.Add("_pragma", "foreign_keys(1)")
	pragmas.Add("_pragma", "busy_timeout(5000)")
	pragmas.Add("_pragma", "journal_mode(WAL)")
	return (&url.URL{Scheme: "file", Path: dsnPath, RawQuery: pragmas.Encode()}).String()
}

// Close releases the underlying SQLite database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) CreateRun(ctx context.Context, run domain.Run) error {
	if err := validateRunID(run.ID); err != nil {
		return err
	}
	run = normalizeRun(run)
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO runs (id, state, profile, task, repo_root, current_stage, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING`,
		run.ID, run.State, run.Profile, run.Task, run.RepoRoot, run.CurrentStage,
		storeTime(run.CreatedAt), storeTime(run.UpdatedAt))
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return ErrRunAlreadyExists
	}
	return nil
}

func (s *SQLiteStore) UpdateRun(ctx context.Context, run domain.Run, eventType string, payload json.RawMessage) (event domain.RunEvent, err error) {
	if err := validateEvent(run.ID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	run = normalizeRun(run)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RunEvent{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	result, err := tx.ExecContext(ctx, `
		UPDATE runs SET state = ?, profile = ?, task = ?, repo_root = ?, current_stage = ?, created_at = ?, updated_at = ?
		WHERE id = ?`,
		run.State, run.Profile, run.Task, run.RepoRoot, run.CurrentStage, storeTime(run.CreatedAt), storeTime(run.UpdatedAt), run.ID)
	if err != nil {
		return domain.RunEvent{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return domain.RunEvent{}, err
	}
	if changed == 0 {
		return domain.RunEvent{}, ErrRunNotFound
	}
	event, err = appendEventTx(ctx, tx, run.ID, eventType, payload)
	if err != nil {
		return domain.RunEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.RunEvent{}, err
	}
	return event, nil
}

func (s *SQLiteStore) AppendEvent(ctx context.Context, runID domain.RunID, eventType string, payload json.RawMessage) (event domain.RunEvent, err error) {
	if err := validateEvent(runID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RunEvent{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	exists, err := runExistsTx(ctx, tx, runID)
	if err != nil {
		return domain.RunEvent{}, err
	}
	if !exists {
		return domain.RunEvent{}, ErrRunNotFound
	}
	event, err = appendEventTx(ctx, tx, runID, eventType, payload)
	if err != nil {
		return domain.RunEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.RunEvent{}, err
	}
	return event, nil
}

func (s *SQLiteStore) GetRun(ctx context.Context, runID domain.RunID) (domain.Run, error) {
	if err := validateRunID(runID); err != nil {
		return domain.Run{}, err
	}
	var run domain.Run
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, state, profile, task, repo_root, current_stage, created_at, updated_at
		FROM runs WHERE id = ?`, runID).Scan(
		&run.ID, &run.State, &run.Profile, &run.Task, &run.RepoRoot, &run.CurrentStage, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Run{}, ErrRunNotFound
	}
	if err != nil {
		return domain.Run{}, err
	}
	run.CreatedAt = loadTime(createdAt)
	run.UpdatedAt = loadTime(updatedAt)
	return run, nil
}

func storeTime(value time.Time) int64 {
	if value.IsZero() {
		return zeroTimeMarker
	}
	return value.UnixNano()
}

func loadTime(value int64) time.Time {
	if value == zeroTimeMarker {
		return time.Time{}
	}
	return time.Unix(0, value).UTC()
}

func (s *SQLiteStore) ListEvents(ctx context.Context, runID domain.RunID, fromSequence uint64) ([]domain.RunEvent, error) {
	if err := validateRunID(runID); err != nil {
		return nil, err
	}
	exists, err := s.runExists(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRunNotFound
	}
	if fromSequence > math.MaxInt64 {
		return []domain.RunEvent{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT run_id, sequence, type, at, payload, previous_hash, hash
		FROM run_events WHERE run_id = ? AND sequence >= ? ORDER BY sequence`, runID, int64(fromSequence))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]domain.RunEvent, 0)
	for rows.Next() {
		var event domain.RunEvent
		var at int64
		var payload []byte
		if err := rows.Scan(&event.RunID, &event.Sequence, &event.Type, &at, &payload, &event.PreviousHash, &event.Hash); err != nil {
			return nil, err
		}
		event.At = time.Unix(0, at).UTC()
		event.Payload = cloneJSON(payload)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *SQLiteStore) PutArtifact(ctx context.Context, artifact domain.Artifact) error {
	if err := validateArtifact(artifact); err != nil {
		return err
	}
	artifact = cloneArtifact(artifact)
	exists, err := s.runExists(ctx, artifact.RunID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrRunNotFound
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO artifacts (id, run_id, kind, sha256, content, truncated)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET run_id = excluded.run_id, kind = excluded.kind,
		sha256 = excluded.sha256, content = excluded.content, truncated = excluded.truncated`,
		artifact.ID, artifact.RunID, artifact.Kind, artifact.SHA256, artifact.Content, artifact.Truncated)
	return err
}

func (s *SQLiteStore) runExists(ctx context.Context, runID domain.RunID) (bool, error) {
	var found int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM runs WHERE id = ?", runID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func runExistsTx(ctx context.Context, tx *sql.Tx, runID domain.RunID) (bool, error) {
	var found int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM runs WHERE id = ?", runID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func appendEventTx(ctx context.Context, tx *sql.Tx, runID domain.RunID, eventType string, payload json.RawMessage) (domain.RunEvent, error) {
	var sequence uint64
	var previousHash string
	err := tx.QueryRowContext(ctx, `
		SELECT sequence, hash FROM run_events WHERE run_id = ? ORDER BY sequence DESC LIMIT 1`, runID).Scan(&sequence, &previousHash)
	if errors.Is(err, sql.ErrNoRows) {
		sequence = 0
		previousHash = ""
	} else if err != nil {
		return domain.RunEvent{}, err
	}
	event := newEvent(runID, sequence+1, eventType, payload, previousHash, time.Now())
	_, err = tx.ExecContext(ctx, `
		INSERT INTO run_events (run_id, sequence, type, at, payload, previous_hash, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.RunID, event.Sequence, event.Type, event.At.UnixNano(), []byte(event.Payload), event.PreviousHash, event.Hash)
	if err != nil {
		return domain.RunEvent{}, err
	}
	return cloneEvent(event), nil
}
