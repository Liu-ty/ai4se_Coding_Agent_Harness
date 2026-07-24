package budget

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

type staticClock struct {
	now time.Time
}

func (c staticClock) Now() time.Time { return c.now }

type reentrantClock struct {
	now     time.Time
	tracker *Tracker
}

func (c *reentrantClock) Now() time.Time {
	if c.tracker != nil {
		c.tracker.Snapshot()
	}
	return c.now
}

func TestTrackerStopsAtEachCountLimitWithoutOverIncrement(t *testing.T) {
	startedAt := time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: startedAt}

	tests := []struct {
		name    string
		limits  Limits
		record  func(*Tracker) error
		used    func(Usage) int
		stopErr error
	}{
		{
			name:    "decisions",
			limits:  Limits{MaxDecisions: 2},
			record:  (*Tracker).RecordDecision,
			used:    func(u Usage) int { return u.Decisions },
			stopErr: ErrDecisionBudget,
		},
		{
			name:    "mutations",
			limits:  Limits{MaxMutations: 2},
			record:  (*Tracker).RecordMutation,
			used:    func(u Usage) int { return u.Mutations },
			stopErr: ErrMutationBudget,
		},
		{
			name:    "protocol repairs",
			limits:  Limits{MaxProtocolRepairs: 2},
			record:  (*Tracker).RecordProtocolRepair,
			used:    func(u Usage) int { return u.ProtocolRepairs },
			stopErr: ErrProtocolRepairBudget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := New(tt.limits, clock)
			if err := tt.record(tr); err != nil {
				t.Fatalf("first record: %v", err)
			}
			if err := tt.record(tr); err != nil {
				t.Fatalf("second record: %v", err)
			}
			if err := tt.record(tr); !errors.Is(err, tt.stopErr) {
				t.Fatalf("third record error = %v, want errors.Is(..., %v)", err, tt.stopErr)
			}
			if got := tt.used(tr.Snapshot()); got != 2 {
				t.Fatalf("usage after rejected record = %d, want 2", got)
			}
		})
	}
}

func TestTrackerExhaustsNonPositiveCountLimitsImmediately(t *testing.T) {
	tr := New(Limits{}, &fakeClock{})

	if err := tr.RecordDecision(); !errors.Is(err, ErrDecisionBudget) {
		t.Fatalf("decision error = %v, want decision budget stop", err)
	}
	if err := tr.RecordMutation(); !errors.Is(err, ErrMutationBudget) {
		t.Fatalf("mutation error = %v, want mutation budget stop", err)
	}
	if err := tr.RecordProtocolRepair(); !errors.Is(err, ErrProtocolRepairBudget) {
		t.Fatalf("repair error = %v, want repair budget stop", err)
	}
	if got := tr.Snapshot(); got.Decisions != 0 || got.Mutations != 0 || got.ProtocolRepairs != 0 {
		t.Fatalf("usage after rejected records = %+v, want zero counts", got)
	}
}

func TestTrackerCheckTimeAtWallClockBoundaryWithoutMutation(t *testing.T) {
	startedAt := time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: startedAt}
	tr := New(Limits{MaxDecisions: 1, WallClock: 10 * time.Minute}, clock)

	if got := tr.Snapshot().StartedAt; !got.Equal(startedAt) {
		t.Fatalf("StartedAt = %v, want %v", got, startedAt)
	}
	clock.now = startedAt.Add(10*time.Minute - time.Nanosecond)
	if err := tr.CheckTime(); err != nil {
		t.Fatalf("CheckTime just before boundary: %v", err)
	}
	clock.now = startedAt.Add(10 * time.Minute)
	if err := tr.CheckTime(); !errors.Is(err, ErrWallClockBudget) {
		t.Fatalf("CheckTime at boundary error = %v, want wall clock budget", err)
	}
	clock.now = startedAt.Add(11 * time.Minute)
	if err := tr.CheckTime(); !errors.Is(err, ErrWallClockBudget) {
		t.Fatalf("CheckTime after boundary error = %v, want wall clock budget", err)
	}
	if got := tr.Snapshot(); got.Decisions != 0 || !got.StartedAt.Equal(startedAt) {
		t.Fatalf("usage after failed time checks = %+v, want unchanged snapshot", got)
	}
}

func TestTrackerExhaustsNonPositiveWallClockImmediately(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC)}
	tr := New(Limits{WallClock: 0}, clock)

	if err := tr.CheckTime(); !errors.Is(err, ErrWallClockBudget) {
		t.Fatalf("CheckTime error = %v, want wall clock budget", err)
	}
}

func TestStopErrorsExposeReasonAndSupportErrorsIs(t *testing.T) {
	tests := []struct {
		name    string
		limits  Limits
		exhaust func(*Tracker) error
		cause   error
		reason  StopReason
	}{
		{
			name:   "decision",
			limits: Limits{MaxDecisions: 1},
			exhaust: func(tr *Tracker) error {
				if err := tr.RecordDecision(); err != nil {
					return fmt.Errorf("first decision: %w", err)
				}
				return tr.RecordDecision()
			},
			cause:  ErrDecisionBudget,
			reason: StopReasonDecisionBudget,
		},
		{
			name:   "mutation",
			limits: Limits{MaxMutations: 1},
			exhaust: func(tr *Tracker) error {
				if err := tr.RecordMutation(); err != nil {
					return fmt.Errorf("first mutation: %w", err)
				}
				return tr.RecordMutation()
			},
			cause:  ErrMutationBudget,
			reason: StopReasonMutationBudget,
		},
		{
			name:   "protocol repair",
			limits: Limits{MaxProtocolRepairs: 1},
			exhaust: func(tr *Tracker) error {
				if err := tr.RecordProtocolRepair(); err != nil {
					return fmt.Errorf("first protocol repair: %w", err)
				}
				return tr.RecordProtocolRepair()
			},
			cause:  ErrProtocolRepairBudget,
			reason: StopReasonProtocolRepairBudget,
		},
		{
			name:    "wall clock",
			limits:  Limits{WallClock: 0},
			exhaust: (*Tracker).CheckTime,
			cause:   ErrWallClockBudget,
			reason:  StopReasonWallClockBudget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.exhaust(New(tt.limits, staticClock{}))
			if !errors.Is(err, tt.cause) {
				t.Fatalf("errors.Is(%v, %v) = false", err, tt.cause)
			}
			var stop *StopError
			if !errors.As(err, &stop) {
				t.Fatalf("errors.As(%v, *StopError) = false", err)
			}
			if got := stop.Reason(); got != tt.reason {
				t.Fatalf("stop reason = %q, want %q", got, tt.reason)
			}
		})
	}
}

func TestSnapshotIsAValueSnapshot(t *testing.T) {
	tr := New(Limits{MaxDecisions: 2}, &fakeClock{})
	if err := tr.RecordDecision(); err != nil {
		t.Fatal(err)
	}
	snapshot := tr.Snapshot()
	if err := tr.RecordDecision(); err != nil {
		t.Fatal(err)
	}
	if snapshot.Decisions != 1 {
		t.Fatalf("saved snapshot decisions = %d, want 1", snapshot.Decisions)
	}
}

func TestNewPanicsClearlyForNilClock(t *testing.T) {
	defer func() {
		if got := recover(); got != "budget: nil Clock" {
			t.Fatalf("panic = %#v, want %q", got, "budget: nil Clock")
		}
	}()
	New(Limits{}, nil)
}

func TestCheckTimeDoesNotHoldLockWhileCallingClock(t *testing.T) {
	clock := &reentrantClock{now: time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC)}
	tr := New(Limits{WallClock: time.Hour}, clock)
	clock.tracker = tr

	done := make(chan error, 1)
	go func() { done <- tr.CheckTime() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("CheckTime: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("CheckTime did not return; Clock.Now appears to run while Tracker is locked")
	}
}

func TestTrackerConcurrentRecordAndSnapshotCapsUsage(t *testing.T) {
	tr := New(Limits{MaxDecisions: 100}, staticClock{})
	const recorders = 16
	const attemptsPerRecorder = 20
	const snapshotters = 4
	const snapshotsPerWorker = 100

	errs := make(chan error, recorders*attemptsPerRecorder)
	var workers sync.WaitGroup
	for range recorders {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range attemptsPerRecorder {
				if err := tr.RecordDecision(); err != nil {
					errs <- err
				}
			}
		}()
	}
	for range snapshotters {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range snapshotsPerWorker {
				_ = tr.Snapshot()
			}
		}()
	}
	workers.Wait()
	close(errs)

	for err := range errs {
		if !errors.Is(err, ErrDecisionBudget) {
			t.Fatalf("record error = %v, want decision budget stop", err)
		}
	}
	if got := tr.Snapshot().Decisions; got != 100 {
		t.Fatalf("final decisions = %d, want capped usage 100", got)
	}
}
