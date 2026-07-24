package store_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
	_ "modernc.org/sqlite"
)

type factory func(*testing.T) store.Store

func TestMemoryStoreContract(t *testing.T) {
	contract(t, func(t *testing.T) store.Store { return store.NewMemory() })
}

func TestSQLiteStoreContract(t *testing.T) {
	contract(t, sqliteFactory(t))
}

func contract(t *testing.T, newStore factory) {
	t.Helper()
	s := newStore(t)
	ctx := context.Background()
	run := domain.Run{ID: "run-1", State: domain.StateCreated, Task: "repair"}
	if err := s.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	e1, err := s.AppendEvent(ctx, run.ID, "RunCreated", json.RawMessage(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	e2, err := s.AppendEvent(ctx, run.ID, "StateChanged", json.RawMessage(`{"state":"PREFLIGHT"}`))
	if err != nil {
		t.Fatal(err)
	}
	if e2.PreviousHash != e1.Hash || e2.Sequence != 2 {
		t.Fatalf("broken chain: %#v %#v", e1, e2)
	}
	if got, want := e1.Hash, canonicalHash(e1); got != want {
		t.Fatalf("first event hash = %q, want %q", got, want)
	}
	if got, want := e2.Hash, canonicalHash(e2); got != want {
		t.Fatalf("second event hash = %q, want %q", got, want)
	}
	updated := run
	updated.State = domain.StatePreflight
	updated.CurrentStage = "preflight"
	e3, err := s.UpdateRun(ctx, updated, "StateChanged", json.RawMessage(`{"state":"PREFLIGHT"}`))
	if err != nil {
		t.Fatal(err)
	}
	if e3.Sequence != 3 || e3.PreviousHash != e2.Hash || e3.Hash != canonicalHash(e3) {
		t.Fatalf("UpdateRun event = %#v, want a verified third chained event", e3)
	}

	stored, err := s.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored != updated {
		t.Fatalf("GetRun() = %#v, want %#v", stored, updated)
	}
	events, err := s.ListEvents(ctx, run.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Sequence != 2 || events[1].Sequence != 3 {
		t.Fatalf("ListEvents(from=2) = %#v, want sequences 2 and 3", events)
	}
}

func TestStoresRejectRequiredIdentifiers(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			if err := s.CreateRun(ctx, domain.Run{ID: ""}); !errors.Is(err, store.ErrEmptyRunID) {
				t.Fatalf("empty run ID error = %v", err)
			}
			if err := s.CreateRun(ctx, domain.Run{ID: "run-1"}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.AppendEvent(ctx, "run-1", "", nil); !errors.Is(err, store.ErrEmptyEventType) {
				t.Fatalf("empty event type error = %v", err)
			}
			if err := s.PutArtifact(ctx, domain.Artifact{RunID: "run-1"}); !errors.Is(err, store.ErrEmptyArtifactID) {
				t.Fatalf("empty artifact ID error = %v", err)
			}
		})
	}
}

func TestStoresCopyEventPayloadOnIngressAndEgress(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			if err := s.CreateRun(ctx, domain.Run{ID: "run-1"}); err != nil {
				t.Fatal(err)
			}
			payload := json.RawMessage(`{"safe":true}`)
			if _, err := s.AppendEvent(ctx, "run-1", "Recorded", payload); err != nil {
				t.Fatal(err)
			}
			payload[2] = 'X'
			events, err := s.ListEvents(ctx, "run-1", 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(events[0].Payload); got != `{"safe":true}` {
				t.Fatalf("stored payload mutated through caller: %q", got)
			}
			events[0].Payload[2] = 'Y'
			events, err = s.ListEvents(ctx, "run-1", 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(events[0].Payload); got != `{"safe":true}` {
				t.Fatalf("stored payload mutated through returned event: %q", got)
			}
		})
	}
}

func TestStoresAcceptEmptyPayloadAndArtifactContent(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			if err := s.CreateRun(ctx, domain.Run{ID: "run-1"}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.AppendEvent(ctx, "run-1", "EmptyPayload", nil); err != nil {
				t.Fatal(err)
			}
			if err := s.PutArtifact(ctx, domain.Artifact{ID: "artifact-1", RunID: "run-1"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStoresAllocateConcurrentSequences(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			if err := s.CreateRun(ctx, domain.Run{ID: "run-1"}); err != nil {
				t.Fatal(err)
			}
			const eventCount = 32
			sequences := make(chan uint64, eventCount)
			errs := make(chan error, eventCount)
			var wg sync.WaitGroup
			for range eventCount {
				wg.Add(1)
				go func() {
					defer wg.Done()
					event, err := s.AppendEvent(ctx, "run-1", "Concurrent", json.RawMessage(`{}`))
					if err != nil {
						errs <- err
						return
					}
					sequences <- event.Sequence
				}()
			}
			wg.Wait()
			close(sequences)
			close(errs)
			for err := range errs {
				t.Fatal(err)
			}
			got := make([]int, 0, eventCount)
			for sequence := range sequences {
				got = append(got, int(sequence))
			}
			sort.Ints(got)
			for index, sequence := range got {
				if want := index + 1; sequence != want {
					t.Fatalf("sequences = %v, want 1 through %d", got, eventCount)
				}
			}
		})
	}
}

func TestStoresRollbackRunUpdateWhenEventValidationFails(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			run := domain.Run{ID: "run-1", State: domain.StateCreated}
			if err := s.CreateRun(ctx, run); err != nil {
				t.Fatal(err)
			}
			run.State = domain.StatePreflight
			if _, err := s.UpdateRun(ctx, run, "", json.RawMessage(`{}`)); !errors.Is(err, store.ErrEmptyEventType) {
				t.Fatalf("UpdateRun empty event type error = %v", err)
			}
			got, err := s.GetRun(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.State != domain.StateCreated {
				t.Fatalf("run state = %q after failed update, want %q", got.State, domain.StateCreated)
			}
			events, err := s.ListEvents(ctx, run.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 0 {
				t.Fatalf("events after failed update = %#v, want none", events)
			}
		})
	}
}

func TestSQLiteUpdateRunRollsBackWhenEventInsertFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")
	ctx := context.Background()
	s, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	original := domain.Run{ID: "run-1", State: domain.StateCreated}
	if err := s.CreateRun(ctx, original); err != nil {
		t.Fatal(err)
	}
	first, err := s.AppendEvent(ctx, original.ID, "RunCreated", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if _, err := raw.Exec(`
		CREATE TRIGGER reject_state_change_event
		BEFORE INSERT ON run_events
		WHEN NEW.type = 'StateChanged'
		BEGIN
			SELECT RAISE(ABORT, 'test rejects event insert');
		END;`); err != nil {
		t.Fatal(err)
	}

	updated := original
	updated.State = domain.StatePreflight
	if _, err := s.UpdateRun(ctx, updated, "StateChanged", json.RawMessage(`{"state":"PREFLIGHT"}`)); err == nil {
		t.Fatal("UpdateRun() succeeded despite the event-insert trigger")
	}
	got, err := s.GetRun(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != original {
		t.Fatalf("run after failed transactional update = %#v, want %#v", got, original)
	}
	events, err := s.ListEvents(ctx, original.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Sequence != first.Sequence || events[0].Hash != first.Hash {
		t.Fatalf("events after failed transactional update = %#v, want original event %#v", events, first)
	}
}

func TestSQLiteStorePersistsAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")
	ctx := context.Background()
	s, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	run := domain.Run{ID: "run-1", State: domain.StateCreated, Task: "persist"}
	if err := s.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendEvent(ctx, run.ID, "RunCreated", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = store.OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	got, err := s.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != run {
		t.Fatalf("GetRun after reopen = %#v, want %#v", got, run)
	}
	events, err := s.ListEvents(ctx, run.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Sequence != 1 {
		t.Fatalf("events after reopen = %#v", events)
	}
}

func sqliteFactory(t *testing.T) factory {
	t.Helper()
	return func(t *testing.T) store.Store {
		t.Helper()
		s, err := store.OpenSQLite(filepath.Join(t.TempDir(), "runs.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
}

func memoryFactory() factory {
	return func(*testing.T) store.Store { return store.NewMemory() }
}

func canonicalHash(event domain.RunEvent) string {
	hash := sha256.New()
	writeBytes := func(value []byte) {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write(value)
	}
	var sequence [8]byte
	binary.BigEndian.PutUint64(sequence[:], event.Sequence)
	var at [8]byte
	binary.BigEndian.PutUint64(at[:], uint64(event.At.UnixNano()))
	writeBytes([]byte(event.RunID))
	writeBytes(sequence[:])
	writeBytes([]byte(event.Type))
	writeBytes(at[:])
	writeBytes(event.Payload)
	writeBytes([]byte(event.PreviousHash))
	return hex.EncodeToString(hash.Sum(nil))
}
