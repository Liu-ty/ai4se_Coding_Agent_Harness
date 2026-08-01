package store_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
	_ "modernc.org/sqlite"
)

type factory func(*testing.T) storeport.Store
type clockedFactory func(*testing.T, store.Clock) storeport.Store

func TestMemoryStoreContract(t *testing.T) {
	contract(t, func(t *testing.T) storeport.Store { return store.NewMemory() })
}

func TestSQLiteStoreContract(t *testing.T) {
	contract(t, sqliteFactory(t))
}

func contract(t *testing.T, newStore factory) {
	t.Helper()
	s := newStore(t)
	ctx := context.Background()
	run := domain.Run{
		ID:        "run-1",
		State:     domain.StateCreated,
		Task:      "repair",
		CreatedAt: time.Now(),
		UpdatedAt: time.Date(2026, time.July, 24, 9, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60)),
	}
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
	updated := normalizedRun(run)
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
	if stored != normalizedRun(updated) {
		t.Fatalf("GetRun() = %#v, want %#v", stored, normalizedRun(updated))
	}
	events, err := s.ListEvents(ctx, run.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Sequence != 2 || events[1].Sequence != 3 {
		t.Fatalf("ListEvents(from=2) = %#v, want sequences 2 and 3", events)
	}
	artifact := domain.Artifact{
		ID: "artifact-1", RunID: run.ID, Kind: "diff", SHA256: "digest",
		Content: []byte("content"), Truncated: true,
	}
	if err := s.PutArtifact(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	gotArtifact, err := s.GetArtifact(ctx, run.ID, artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotArtifact.ID != artifact.ID || gotArtifact.RunID != artifact.RunID ||
		gotArtifact.Kind != artifact.Kind || gotArtifact.SHA256 != artifact.SHA256 ||
		string(gotArtifact.Content) != string(artifact.Content) ||
		gotArtifact.Truncated != artifact.Truncated {
		t.Fatalf("GetArtifact() = %#v, want %#v", gotArtifact, artifact)
	}
	gotArtifact.Content[0] = 'X'
	gotArtifact, err = s.GetArtifact(ctx, run.ID, artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotArtifact.Content) != "content" {
		t.Fatalf("artifact content aliases store memory: %q", gotArtifact.Content)
	}
	if _, err := s.GetArtifact(ctx, run.ID, "missing"); !errors.Is(err, storeport.ErrArtifactNotFound) {
		t.Fatalf("missing GetArtifact() error = %v", err)
	}
}

func TestStoresRejectDuplicateRunWithoutMutation(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			original := domain.Run{ID: "run-1", State: domain.StateCreated, Task: "original"}
			if err := s.CreateRun(ctx, original); err != nil {
				t.Fatal(err)
			}
			event, err := s.AppendEvent(ctx, original.ID, "RunCreated", json.RawMessage(`{}`))
			if err != nil {
				t.Fatal(err)
			}
			if err := s.CreateRun(ctx, domain.Run{ID: original.ID, State: domain.StateStopped, Task: "replacement"}); !errors.Is(err, storeport.ErrRunAlreadyExists) {
				t.Fatalf("duplicate CreateRun() error = %v", err)
			}
			got, err := s.GetRun(ctx, original.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got != original {
				t.Fatalf("run after duplicate CreateRun() = %#v, want %#v", got, original)
			}
			events, err := s.ListEvents(ctx, original.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].Hash != event.Hash {
				t.Fatalf("events after duplicate CreateRun() = %#v, want %#v", events, event)
			}
		})
	}
}

func TestStoresRejectArtifactsForMissingRuns(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			err := newStore(t).PutArtifact(context.Background(), domain.Artifact{ID: "artifact-1", RunID: "missing"})
			if !errors.Is(err, storeport.ErrRunNotFound) {
				t.Fatalf("PutArtifact() missing run error = %v", err)
			}
		})
	}
}

func TestStoresRejectArtifactIDReuseAcrossRunsWithoutReplacingOriginal(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			for _, id := range []domain.RunID{"run-a", "run-b"} {
				if err := s.CreateRun(ctx, domain.Run{ID: id, State: domain.StateCreated}); err != nil {
					t.Fatal(err)
				}
			}
			original := domain.Artifact{ID: "shared", RunID: "run-a", Content: []byte("original")}
			if err := s.PutArtifact(ctx, original); err != nil {
				t.Fatal(err)
			}
			if err := s.PutArtifact(ctx, domain.Artifact{
				ID: "shared", RunID: "run-b", Content: []byte("replacement"),
			}); !errors.Is(err, storeport.ErrArtifactExists) {
				t.Fatalf("cross-run duplicate error = %v, want ErrArtifactExists", err)
			}
			got, err := s.GetArtifact(ctx, "run-a", "shared")
			if err != nil || string(got.Content) != "original" {
				t.Fatalf("original artifact = %#v, %v", got, err)
			}
			if _, err := s.GetArtifact(ctx, "run-b", "shared"); !errors.Is(err, storeport.ErrArtifactNotFound) {
				t.Fatalf("second run artifact error = %v", err)
			}
		})
	}
}

func TestStoresListEventsReturnsNonNilEmptySlice(t *testing.T) {
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
			events, err := s.ListEvents(ctx, "run-1", math.MaxUint64)
			if err != nil {
				t.Fatal(err)
			}
			if events == nil || len(events) != 0 {
				t.Fatalf("ListEvents() = %#v, want non-nil empty slice", events)
			}
		})
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
			if err := s.CreateRun(ctx, domain.Run{ID: ""}); !errors.Is(err, storeport.ErrEmptyRunID) {
				t.Fatalf("empty run ID error = %v", err)
			}
			if err := s.CreateRun(ctx, domain.Run{ID: "run-1"}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.AppendEvent(ctx, "run-1", "", nil); !errors.Is(err, storeport.ErrEmptyEventType) {
				t.Fatalf("empty event type error = %v", err)
			}
			if err := s.PutArtifact(ctx, domain.Artifact{RunID: "run-1"}); !errors.Is(err, storeport.ErrEmptyArtifactID) {
				t.Fatalf("empty artifact ID error = %v", err)
			}
			if _, err := s.GetArtifact(ctx, "run-1", ""); !errors.Is(err, storeport.ErrEmptyArtifactID) {
				t.Fatalf("empty GetArtifact ID error = %v", err)
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
			events, err := s.ListEvents(ctx, "run-1", 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != eventCount {
				t.Fatalf("event count = %d, want %d", len(events), eventCount)
			}
			for index, event := range events {
				if event.Sequence != uint64(index+1) || event.Hash != canonicalHash(event) {
					t.Fatalf("invalid concurrent event at index %d: %#v", index, event)
				}
				previousHash := ""
				if index > 0 {
					previousHash = events[index-1].Hash
				}
				if event.PreviousHash != previousHash {
					t.Fatalf("event %d previous hash = %q, want %q", event.Sequence, event.PreviousHash, previousHash)
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
			if _, err := s.UpdateRun(ctx, run, "", json.RawMessage(`{}`)); !errors.Is(err, storeport.ErrEmptyEventType) {
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

func TestStoresHonorAlreadyCancelledContextsWithoutMutation(t *testing.T) {
	for name, newStore := range map[string]factory{
		"memory": memoryFactory(),
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			initial := domain.Run{ID: "run-1", State: domain.StateCreated, Task: "initial"}
			if err := s.CreateRun(cancelled, initial); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled CreateRun() error = %v, want context.Canceled", err)
			}
			if _, err := s.GetRun(context.Background(), initial.ID); !errors.Is(err, storeport.ErrRunNotFound) {
				t.Fatalf("run after cancelled CreateRun() error = %v, want ErrRunNotFound", err)
			}

			ctx := context.Background()
			if err := s.CreateRun(ctx, initial); err != nil {
				t.Fatal(err)
			}
			first, err := s.AppendEvent(ctx, initial.ID, "RunCreated", json.RawMessage(`{}`))
			if err != nil {
				t.Fatal(err)
			}

			updated := initial
			updated.State = domain.StatePreflight
			if _, err := s.UpdateRun(cancelled, updated, "StateChanged", json.RawMessage(`{"state":"PREFLIGHT"}`)); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled UpdateRun() error = %v, want context.Canceled", err)
			}
			got, err := s.GetRun(ctx, initial.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got != initial {
				t.Fatalf("run after cancelled UpdateRun() = %#v, want %#v", got, initial)
			}

			if _, err := s.AppendEvent(cancelled, initial.ID, "ShouldNotRecord", json.RawMessage(`{}`)); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled AppendEvent() error = %v, want context.Canceled", err)
			}
			events, err := s.ListEvents(ctx, initial.ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].Hash != first.Hash {
				t.Fatalf("events after cancelled writes = %#v, want only %#v", events, first)
			}
			if _, err := s.GetRun(cancelled, initial.ID); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled GetRun() error = %v, want context.Canceled", err)
			}
			if _, err := s.ListEvents(cancelled, initial.ID, 0); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled ListEvents() error = %v, want context.Canceled", err)
			}
			if err := s.PutArtifact(cancelled, domain.Artifact{ID: "artifact-1", RunID: initial.ID}); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled PutArtifact() error = %v, want context.Canceled", err)
			}
			if _, err := s.GetArtifact(cancelled, initial.ID, "artifact-1"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled GetArtifact() error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestStoresUseInjectedClockForEventTimestampsAndHashes(t *testing.T) {
	firstAt := time.Date(2026, time.July, 24, 1, 2, 3, 4, time.FixedZone("UTC+8", 8*60*60))
	secondAt := time.Date(2026, time.July, 24, 1, 2, 4, 5, time.FixedZone("UTC+8", 8*60*60))
	thirdAt := time.Date(2026, time.July, 24, 1, 2, 5, 6, time.FixedZone("UTC+8", 8*60*60))
	for name, newStore := range map[string]clockedFactory{
		"memory": memoryClockFactory(),
		"sqlite": sqliteClockFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			clock := &scriptedClock{times: []time.Time{firstAt, secondAt, thirdAt}}
			s := newStore(t, clock)
			ctx := context.Background()
			run := domain.Run{ID: "run-1", State: domain.StateCreated}
			if err := s.CreateRun(ctx, run); err != nil {
				t.Fatal(err)
			}
			e1, err := s.AppendEvent(ctx, run.ID, "RunCreated", json.RawMessage(`{"n":1}`))
			if err != nil {
				t.Fatal(err)
			}
			e2, err := s.AppendEvent(ctx, run.ID, "Observed", json.RawMessage(`{"n":2}`))
			if err != nil {
				t.Fatal(err)
			}
			run.State = domain.StatePreflight
			e3, err := s.UpdateRun(ctx, run, "StateChanged", json.RawMessage(`{"n":3}`))
			if err != nil {
				t.Fatal(err)
			}
			wantTimes := []time.Time{firstAt.UTC(), secondAt.UTC(), thirdAt.UTC()}
			for index, event := range []domain.RunEvent{e1, e2, e3} {
				if !event.At.Equal(wantTimes[index]) {
					t.Fatalf("event %d At = %s, want %s", index+1, event.At, wantTimes[index])
				}
				if event.Hash != canonicalHash(event) {
					t.Fatalf("event %d hash = %q, want canonical hash %q", index+1, event.Hash, canonicalHash(event))
				}
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
	if _, err := raw.ExecContext(ctx, `
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
	return func(t *testing.T) storeport.Store {
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
	return func(*testing.T) storeport.Store { return store.NewMemory() }
}

func sqliteClockFactory(t *testing.T) clockedFactory {
	t.Helper()
	return func(t *testing.T, clock store.Clock) storeport.Store {
		t.Helper()
		s, err := store.OpenSQLiteWithClock(filepath.Join(t.TempDir(), "runs.db"), clock)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
}

func memoryClockFactory() clockedFactory {
	return func(_ *testing.T, clock store.Clock) storeport.Store {
		return store.NewMemoryWithClock(clock)
	}
}

type scriptedClock struct {
	mu    sync.Mutex
	times []time.Time
}

func (c *scriptedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.times) == 0 {
		panic("scripted clock exhausted")
	}
	now := c.times[0]
	c.times = c.times[1:]
	return now
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

func normalizedRun(run domain.Run) domain.Run {
	run.CreatedAt = normalizedTime(run.CreatedAt)
	run.UpdatedAt = normalizedTime(run.UpdatedAt)
	return run
}

func normalizedTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC()
}
