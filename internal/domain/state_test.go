package domain_test

import (
	"errors"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func TestRepairFlowAllowsBaselineToDecision(t *testing.T) {
	if err := domain.Transition(domain.StateBaselineValidating, domain.StateDeciding); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
}

func TestDecisionCannotClaimSuccess(t *testing.T) {
	err := domain.Transition(domain.StateDeciding, domain.StateSucceeded)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestReviewFlowEndsWithoutValidation(t *testing.T) {
	if err := domain.Transition(domain.StateDeciding, domain.StateReviewComplete); err != nil {
		t.Fatalf("expected review completion: %v", err)
	}
}

func TestTerminalStateRejectsTransition(t *testing.T) {
	terminalStates := []domain.RunState{
		domain.StateSucceeded,
		domain.StateReviewComplete,
		domain.StateStopped,
	}
	for _, ts := range terminalStates {
		err := domain.Transition(ts, domain.StateDeciding)
		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("terminal state %s: expected ErrInvalidTransition, got %v", ts, err)
		}
	}
}

func TestFullRepairFlowHappyPath(t *testing.T) {
	flow := []domain.RunState{
		domain.StateCreated,
		domain.StatePreflight,
		domain.StateBaselineValidating,
		domain.StateDeciding,
		domain.StateAwaitingApproval,
		domain.StateExecuting,
		domain.StateValidating,
		domain.StateFinalValidating,
		domain.StateSucceeded,
	}
	for i := 0; i < len(flow)-1; i++ {
		from, to := flow[i], flow[i+1]
		if err := domain.Transition(from, to); err != nil {
			t.Fatalf("expected valid transition %s → %s: %v", from, to, err)
		}
	}
}

func TestPreflightToDeciding(t *testing.T) {
	if err := domain.Transition(domain.StatePreflight, domain.StateDeciding); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
}

func TestDecidingToStopped(t *testing.T) {
	if err := domain.Transition(domain.StateDeciding, domain.StateStopped); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
}

func TestValidatingToStopped(t *testing.T) {
	if err := domain.Transition(domain.StateValidating, domain.StateStopped); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
}

func TestFinalValidatingDeciding(t *testing.T) {
	if err := domain.Transition(domain.StateFinalValidating, domain.StateDeciding); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
}

func TestSelfTransitionRejected(t *testing.T) {
	err := domain.Transition(domain.StateDeciding, domain.StateDeciding)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}
