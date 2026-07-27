// Package budget enforces the resource limits of one harness run.
package budget

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrDecisionBudget       = errors.New("decision budget exhausted")
	ErrMutationBudget       = errors.New("mutation budget exhausted")
	ErrProtocolRepairBudget = errors.New("protocol repair budget exhausted")
	ErrWallClockBudget      = errors.New("wall-clock budget exhausted")
)

// StopReason identifies the budget that prevented further run progress.
type StopReason string

const (
	StopReasonDecisionBudget       StopReason = "decision_budget"
	StopReasonMutationBudget       StopReason = "mutation_budget"
	StopReasonProtocolRepairBudget StopReason = "protocol_repair_budget"
	StopReasonWallClockBudget      StopReason = "wall_clock_budget"
)

// StopError is returned when a budget has been exhausted.
// Its Reason provides stable programmatic inspection and its cause supports errors.Is.
type StopError struct {
	reason StopReason
	cause  error
}

func (e *StopError) Error() string {
	return fmt.Sprintf("budget stopped: %s", e.reason)
}

func (e *StopError) Unwrap() error { return e.cause }

// Reason returns the immutable reason the tracker stopped.
func (e *StopError) Reason() StopReason { return e.reason }

func newStopError(reason StopReason, cause error) *StopError {
	return &StopError{reason: reason, cause: cause}
}

// Limits defines the maximum resources available to a single run.
type Limits struct {
	MaxDecisions       int
	MaxMutations       int
	MaxProtocolRepairs int
	WallClock          time.Duration
}

// Usage is a value snapshot of a tracker's resource consumption.
// ProtocolRepairs counts repairs at the current decision point, not across the run.
type Usage struct {
	Decisions       int
	Mutations       int
	ProtocolRepairs int
	StartedAt       time.Time
}

// Clock supplies the current time, allowing callers to control time in tests.
type Clock interface {
	Now() time.Time
}

// Tracker records consumption and rejects attempts that would exceed its limits.
type Tracker struct {
	mu     sync.Mutex
	limits Limits
	usage  Usage
	// startedAt duplicates usage.StartedAt so CheckTime can read it without
	// t.mu; a custom Clock.Now may reenter Tracker and deadlock otherwise.
	startedAt time.Time
	clock     Clock
}

// New creates a tracker beginning at the clock's current time.
// clock must be non-nil.
func New(limits Limits, clock Clock) *Tracker {
	if clock == nil {
		panic("budget: nil Clock")
	}
	startedAt := clock.Now()
	return &Tracker{
		limits:    limits,
		usage:     Usage{StartedAt: startedAt},
		startedAt: startedAt,
		clock:     clock,
	}
}

// RecordDecision records one decision when the decision budget permits it.
func (t *Tracker) RecordDecision() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.record(&t.usage.Decisions, t.limits.MaxDecisions, StopReasonDecisionBudget, ErrDecisionBudget); err != nil {
		return err
	}
	t.usage.ProtocolRepairs = 0
	return nil
}

// RecordMutation records one mutation when the mutation budget permits it.
func (t *Tracker) RecordMutation() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.record(&t.usage.Mutations, t.limits.MaxMutations, StopReasonMutationBudget, ErrMutationBudget)
}

// RecordProtocolRepair records one protocol repair when its budget permits it.
func (t *Tracker) RecordProtocolRepair() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.record(&t.usage.ProtocolRepairs, t.limits.MaxProtocolRepairs, StopReasonProtocolRepairBudget, ErrProtocolRepairBudget)
}

func (t *Tracker) record(used *int, limit int, reason StopReason, cause error) error {
	if limit <= 0 || *used >= limit {
		return newStopError(reason, cause)
	}
	*used++
	return nil
}

// CheckTime returns a stop error once the wall-clock budget has elapsed.
// It intentionally avoids t.mu so reentrant Clock implementations cannot
// self-deadlock while checking time.
func (t *Tracker) CheckTime() error {
	now := t.clock.Now()
	if t.limits.WallClock <= 0 || now.Sub(t.startedAt) >= t.limits.WallClock {
		return newStopError(StopReasonWallClockBudget, ErrWallClockBudget)
	}
	return nil
}

// Snapshot returns a copy of the current usage.
func (t *Tracker) Snapshot() Usage {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.usage
}
