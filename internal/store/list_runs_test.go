package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

func TestStoresCreateRunAndInitialEventAtomically(t *testing.T) {
	for name, makeStore := range map[string]func(*testing.T) storeport.Store{
		"memory": func(*testing.T) storeport.Store { return store.NewMemory() },
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := makeStore(t)
			run := domain.Run{ID: "run-atomic", State: domain.StateCreated}
			event, err := s.CreateRunWithEvent(
				context.Background(), run, "RunCreated", json.RawMessage(`{"provider":"mock"}`),
			)
			if err != nil {
				t.Fatal(err)
			}
			if event.Sequence != 1 || event.Type != "RunCreated" {
				t.Fatalf("event = %#v", event)
			}
			events, err := s.ListEvents(context.Background(), run.ID, 1)
			if err != nil || len(events) != 1 || events[0].Hash != event.Hash {
				t.Fatalf("events = %#v, %v", events, err)
			}

			missing := domain.Run{ID: "must-not-exist", State: domain.StateCreated}
			if _, err := s.CreateRunWithEvent(context.Background(), missing, "", nil); !errors.Is(err, storeport.ErrEmptyEventType) {
				t.Fatalf("invalid-event error = %v", err)
			}
			if _, err := s.GetRun(context.Background(), missing.ID); !errors.Is(err, storeport.ErrRunNotFound) {
				t.Fatalf("partially created run error = %v", err)
			}
		})
	}
}

func TestStoresConditionallyUpdateLifecycleState(t *testing.T) {
	for name, makeStore := range map[string]func(*testing.T) storeport.Store{
		"memory": func(*testing.T) storeport.Store { return store.NewMemory() },
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := makeStore(t)
			run := domain.Run{ID: "run-cas", State: domain.StateCreated}
			if err := s.CreateRun(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			run.State = domain.StatePreflight
			if _, err := s.UpdateRunIfState(
				context.Background(), run, domain.StateCreated, "PreflightStarted", json.RawMessage(`{}`),
			); err != nil {
				t.Fatal(err)
			}
			stale := run
			stale.State = domain.StateStopped
			if _, err := s.UpdateRunIfState(
				context.Background(), stale, domain.StateCreated, "RunStopped", json.RawMessage(`{}`),
			); !errors.Is(err, storeport.ErrRunStateChanged) {
				t.Fatalf("stale update error = %v", err)
			}
			got, err := s.GetRun(context.Background(), run.ID)
			if err != nil || got.State != domain.StatePreflight {
				t.Fatalf("run = %#v, %v", got, err)
			}
			events, err := s.ListEvents(context.Background(), run.ID, 1)
			if err != nil || len(events) != 1 {
				t.Fatalf("events after stale update = %#v, %v", events, err)
			}
		})
	}
}

func TestStoresListRunsInStableIDOrder(t *testing.T) {
	for name, makeStore := range map[string]func(*testing.T) storeport.Store{
		"memory": func(*testing.T) storeport.Store { return store.NewMemory() },
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := makeStore(t)
			empty, err := s.ListRuns(context.Background())
			if err != nil || empty == nil || len(empty) != 0 {
				t.Fatalf("empty ListRuns() = %#v, %v; want non-nil empty slice", empty, err)
			}
			for _, id := range []domain.RunID{"run-b", "run-a"} {
				if err := s.CreateRun(context.Background(), domain.Run{ID: id}); err != nil {
					t.Fatal(err)
				}
			}
			runs, err := s.ListRuns(context.Background())
			if err != nil || len(runs) != 2 || runs[0].ID != "run-a" || runs[1].ID != "run-b" {
				t.Fatalf("ListRuns() = %#v, %v", runs, err)
			}
		})
	}
}
