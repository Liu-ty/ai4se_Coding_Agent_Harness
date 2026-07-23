package domain

// allowedTransitions maps each state to the set of states it may transition to.
// Terminal states (SUCCEEDED, REVIEW_COMPLETE, STOPPED) are present with empty
// maps to make the intent explicit: no outgoing transitions are permitted.
var allowedTransitions = map[RunState]map[RunState]bool{
	StateCreated:            {StatePreflight: true},
	StatePreflight:          {StateBaselineValidating: true, StateDeciding: true, StateStopped: true},
	StateBaselineValidating: {StateDeciding: true, StateStopped: true},
	StateDeciding:           {StateAwaitingApproval: true, StateExecuting: true, StateReviewComplete: true, StateStopped: true},
	StateAwaitingApproval:   {StateExecuting: true, StateDeciding: true, StateStopped: true},
	StateExecuting:          {StateDeciding: true, StateValidating: true, StateStopped: true},
	StateValidating:         {StateDeciding: true, StateFinalValidating: true, StateStopped: true},
	StateFinalValidating:    {StateSucceeded: true, StateDeciding: true, StateStopped: true},
	StateSucceeded:          {},
	StateReviewComplete:     {},
	StateStopped:            {},
}

// Transition checks whether the state transition from → to is valid.
// It returns ErrInvalidTransition if the transition is not permitted.
func Transition(from, to RunState) error {
	if !allowedTransitions[from][to] {
		return ErrInvalidTransition
	}
	return nil
}
