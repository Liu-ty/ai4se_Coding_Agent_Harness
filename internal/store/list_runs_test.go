package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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

func TestStoresListRunsByRecentUpdateWithStablePaginationAndFiltering(t *testing.T) {
	for name, makeStore := range map[string]func(*testing.T) storeport.Store{
		"memory": func(*testing.T) storeport.Store { return store.NewMemory() },
		"sqlite": sqliteFactory(t),
	} {
		t.Run(name, func(t *testing.T) {
			s := makeStore(t)
			empty, err := s.ListRuns(context.Background(), storeport.RunListQuery{Limit: 2})
			if err != nil || empty.Runs == nil || len(empty.Runs) != 0 || empty.HasMore {
				t.Fatalf("empty ListRuns() = %#v, %v; want non-nil empty page", empty, err)
			}
			updated := time.Unix(1700000000, 0).UTC()
			for _, run := range []domain.Run{
				{ID: "run-b", UpdatedAt: updated},
				{ID: "run-old", UpdatedAt: updated.Add(-time.Hour)},
				{ID: "run-a", UpdatedAt: updated},
				{ID: "run-new", UpdatedAt: updated.Add(time.Hour)},
			} {
				if err := s.CreateRun(context.Background(), run); err != nil {
					t.Fatal(err)
				}
			}
			page, err := s.ListRuns(context.Background(), storeport.RunListQuery{Limit: 2, Offset: 1})
			if err != nil || len(page.Runs) != 2 ||
				page.Runs[0].ID != "run-a" || page.Runs[1].ID != "run-b" || !page.HasMore {
				t.Fatalf("ListRuns() = %#v, %v", page, err)
			}
			fixed := []domain.RunID{"run-old", "run-a"}
			filtered, err := s.ListRuns(context.Background(), storeport.RunListQuery{
				Limit: 1, Offset: 1, IDs: fixed,
			})
			if err != nil || len(filtered.Runs) != 1 || filtered.Runs[0].ID != "run-old" || filtered.HasMore {
				t.Fatalf("filtered ListRuns() = %#v, %v", filtered, err)
			}
		})
	}
}
