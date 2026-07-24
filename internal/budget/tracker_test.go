package budget

import (
	"errors"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

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
	tr := New(Limits{MaxDecisions: 1}, &fakeClock{})
	if err := tr.RecordDecision(); err != nil {
		t.Fatal(err)
	}
	err := tr.RecordDecision()
	if !errors.Is(err, ErrDecisionBudget) {
		t.Fatalf("errors.Is(%v, ErrDecisionBudget) = false", err)
	}
	var stop *StopError
	if !errors.As(err, &stop) {
		t.Fatalf("errors.As(%v, *StopError) = false", err)
	}
	if stop.Reason != StopReasonDecisionBudget {
		t.Fatalf("stop reason = %q, want %q", stop.Reason, StopReasonDecisionBudget)
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
