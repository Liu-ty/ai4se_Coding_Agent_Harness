package agent

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/budget"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

var (
	ErrApprovalNotGranted        = errors.New("approval is not granted for the pending action")
	ErrApprovalRejectionMismatch = errors.New("approval rejection does not match the pending action")
)

type ActionResult struct {
	DiffDigest  string
	Observation domain.Observation
}
type ActionExecutor interface {
	Execute(context.Context, domain.Action) (ActionResult, error)
}
type ValidationResult struct {
	StageID     string
	Passed      bool
	Observation domain.Observation
}
type Validator interface {
	Baseline(context.Context, domain.Run) ValidationResult
	Current(context.Context, domain.Run) ValidationResult
	Final(context.Context, domain.Run) ValidationResult
}
type Dependencies struct {
	Store      storeport.Store
	Provider   provider.Provider
	Actions    ActionExecutor
	Policy     policy.Engine
	Feedback   feedback.Pipeline
	Validation Validator
	Budget     *budget.Tracker
	Progress   *budget.ProgressDetector
	Approvals  *policy.ApprovalStore
	Context    ContextAssembler
	Baselines  map[string]string
}
type pendingApproval struct {
	run    domain.Run
	action domain.Action
	digest policy.ApprovalDigest
}
type Loop struct {
	d       Dependencies
	mu      sync.Mutex
	pending map[domain.RunID]pendingApproval
}
type Result struct {
	State    domain.RunState
	StopCode string
}

func New(d Dependencies) *Loop { return &Loop{d: d, pending: make(map[domain.RunID]pendingApproval)} }

// BindBaselines replaces the immutable snapshot incorporated into future
// approval digests. Composition calls this before the loop starts.
func (l *Loop) BindBaselines(baselines map[string]string) {
	copyOfBaselines := make(map[string]string, len(baselines))
	for path, hash := range baselines {
		copyOfBaselines[path] = hash
	}
	l.mu.Lock()
	l.d.Baselines = copyOfBaselines
	l.mu.Unlock()
}

func (l *Loop) Run(ctx context.Context, run domain.Run) (Result, error) {
	return l.run(ctx, run, nil)
}
func (l *Loop) run(ctx context.Context, run domain.Run, last *domain.StructuredFeedback) (Result, error) {
	if ctx.Err() != nil {
		return l.stop(ctx, &run, "USER_CANCELLED")
	}
	if run.State == domain.StateCreated {
		if err := l.transition(ctx, &run, domain.StatePreflight, "PreflightComplete"); err != nil {
			return Result{}, err
		}
	}
	if run.State == domain.StatePreflight && run.Profile != domain.ProfileReview {
		if err := l.transition(ctx, &run, domain.StateBaselineValidating, "BaselineStarted"); err != nil {
			return Result{}, err
		}
		base := l.d.Validation.Baseline(ctx, run)
		if base.Passed {
			return l.stop(ctx, &run, "NO_REPRODUCTION")
		}
		if err := l.event(ctx, run, "BaselineFailed", base); err != nil {
			return Result{}, err
		}
	}
	if run.State != domain.StateDeciding {
		if err := l.transition(ctx, &run, domain.StateDeciding, "Deciding"); err != nil {
			return Result{}, err
		}
	}
	for {
		if err := ctx.Err(); err != nil {
			return l.stop(ctx, &run, "USER_CANCELLED")
		}
		if err := l.d.Budget.CheckTime(); err != nil {
			return l.stop(ctx, &run, "BUDGET_EXHAUSTED")
		}
		response, err := l.d.Provider.Decide(ctx, provider.Request{Task: run.Task, LastFeedback: last})
		if err != nil || !valid(response.Decision) {
			if err := l.d.Budget.RecordProtocolRepair(); err != nil {
				return l.stop(ctx, &run, "PROTOCOL_EXHAUSTED")
			}
			if e := l.event(ctx, run, "ProtocolError", err); e != nil {
				return Result{}, e
			}
			continue
		}
		if err := l.d.Budget.RecordDecision(); err != nil {
			return l.stop(ctx, &run, "BUDGET_EXHAUSTED")
		}
		action := response.Decision.Action
		l.mu.Lock()
		baselines := make(map[string]string, len(l.d.Baselines))
		for path, hash := range l.d.Baselines {
			baselines[path] = hash
		}
		l.mu.Unlock()
		decision := l.d.Policy.Evaluate(policy.Context{
			RunID: run.ID, Profile: run.Profile, Baselines: baselines,
		}, action)
		if decision.Verdict == policy.Deny {
			last = l.feedback(run, "POLICY_DENIED", domain.Observation{Stdout: decision.Message})
			if err := l.event(ctx, run, "PolicyDenied", decision); err != nil {
				return Result{}, err
			}
			if err := l.event(ctx, run, "FeedbackProduced", last); err != nil {
				return Result{}, err
			}
			continue
		}
		if decision.Verdict == policy.RequireApproval {
			request := newApprovalRequired(run.Profile, action, decision, baselines)
			pendingRun := run
			pendingRun.State = domain.StateAwaitingApproval
			l.mu.Lock()
			l.pending[run.ID] = pendingApproval{
				run: pendingRun, action: action, digest: decision.Digest,
			}
			l.mu.Unlock()
			result, err := l.await(ctx, &run, request)
			if err != nil {
				l.deletePending(run.ID, decision.Digest)
				return Result{}, err
			}
			return result, nil
		}
		if err := l.transition(ctx, &run, domain.StateExecuting, "ActionExecuting"); err != nil {
			return Result{}, err
		}
		if action.Kind == "finish" {
			if run.Profile == domain.ProfileReview {
				if err := l.transition(ctx, &run, domain.StateDeciding, "ReviewFinish"); err != nil {
					return Result{}, err
				}
				if err := l.transition(ctx, &run, domain.StateReviewComplete, "ReviewComplete"); err != nil {
					return Result{}, err
				}
				return Result{State: run.State}, nil
			}
			if err := l.transition(ctx, &run, domain.StateValidating, "FinishRequested"); err != nil {
				return Result{}, err
			}
			final := l.d.Validation.Final(ctx, run)
			if !final.Passed {
				last = l.feedback(run, "REGRESSION", final.Observation)
				if err := l.event(ctx, run, "ValidationFailed", final); err != nil {
					return Result{}, err
				}
				if err := l.event(ctx, run, "FeedbackProduced", last); err != nil {
					return Result{}, err
				}
				if err := l.transition(ctx, &run, domain.StateDeciding, "FinalRegression"); err != nil {
					return Result{}, err
				}
				continue
			}
			if err := l.transition(ctx, &run, domain.StateFinalValidating, "FinalValidationPassed"); err != nil {
				return Result{}, err
			}
			if err := l.transition(ctx, &run, domain.StateSucceeded, "RunSucceeded"); err != nil {
				return Result{}, err
			}
			return Result{State: run.State}, nil
		}
		actionResult, err := l.d.Actions.Execute(ctx, action)
		if err != nil {
			last = l.feedback(run, "ACTION_FAILED", domain.Observation{Stdout: err.Error()})
			if e := l.event(ctx, run, "ActionFailed", err); e != nil {
				return Result{}, e
			}
			if e := l.transition(ctx, &run, domain.StateDeciding, "ActionFeedback"); e != nil {
				return Result{}, e
			}
			continue
		}
		if action.Kind == "apply_patch" || action.Kind == "create_file" {
			if err := l.d.Budget.RecordMutation(); err != nil {
				return l.stop(ctx, &run, "BUDGET_EXHAUSTED")
			}
			if err := l.transition(ctx, &run, domain.StateValidating, "ValidationStarted"); err != nil {
				return Result{}, err
			}
			current := l.d.Validation.Current(ctx, run)
			if !current.Passed {
				last = l.feedback(run, "", current.Observation)
				if err := l.event(ctx, run, "ValidationFailed", current); err != nil {
					return Result{}, err
				}
				if err := l.event(ctx, run, "FeedbackProduced", last); err != nil {
					return Result{}, err
				}
				if err := l.transition(ctx, &run, domain.StateDeciding, "ValidationFeedback"); err != nil {
					return Result{}, err
				}
				if l.d.Progress != nil && l.d.Progress.Observe(last.Fingerprint, actionResult.DiffDigest) == budget.ProgressStop {
					return l.stop(ctx, &run, "NO_PROGRESS")
				}
				continue
			}
			if err := l.event(ctx, run, "ValidationPassed", current); err != nil {
				return Result{}, err
			}
			last = &domain.StructuredFeedback{Category: "VALIDATION_PASSED", StageID: current.StageID}
		}
		if err := l.transition(ctx, &run, domain.StateDeciding, "ActionComplete"); err != nil {
			return Result{}, err
		}
	}
}
func (l *Loop) ResumeApproval(ctx context.Context, runID domain.RunID, digest policy.ApprovalDigest) (Result, error) {
	l.mu.Lock()
	pending, ok := l.pending[runID]
	l.mu.Unlock()
	if !ok || pending.digest != digest || l.d.Approvals == nil || !l.d.Approvals.Consume(digest) {
		return Result{}, ErrApprovalNotGranted
	}
	if err := l.transition(ctx, &pending.run, domain.StateExecuting, "ApprovalGranted"); err != nil {
		return Result{}, err
	}
	if _, err := l.d.Actions.Execute(ctx, pending.action); err != nil {
		return Result{}, err
	}
	if pending.action.Kind == "apply_patch" || pending.action.Kind == "create_file" {
		if err := l.d.Budget.RecordMutation(); err != nil {
			return l.stop(ctx, &pending.run, "BUDGET_EXHAUSTED")
		}
		if err := l.transition(ctx, &pending.run, domain.StateValidating, "ValidationStarted"); err != nil {
			return Result{}, err
		}
		current := l.d.Validation.Current(ctx, pending.run)
		if !current.Passed {
			last := l.feedback(pending.run, "", current.Observation)
			if err := l.event(ctx, pending.run, "ValidationFailed", current); err != nil {
				return Result{}, err
			}
			if err := l.event(ctx, pending.run, "FeedbackProduced", last); err != nil {
				return Result{}, err
			}
			if err := l.transition(ctx, &pending.run, domain.StateDeciding, "ValidationFeedback"); err != nil {
				return Result{}, err
			}
			l.deletePending(runID, digest)
			return l.run(ctx, pending.run, last)
		}
		if err := l.event(ctx, pending.run, "ValidationPassed", current); err != nil {
			return Result{}, err
		}
	}
	l.mu.Lock()
	delete(l.pending, runID)
	l.mu.Unlock()
	if pending.run.State == domain.StateValidating {
		if err := l.transition(ctx, &pending.run, domain.StateDeciding, "ActionComplete"); err != nil {
			return Result{}, err
		}
	}
	return l.Run(ctx, pending.run)
}
func (l *Loop) RejectApproval(
	ctx context.Context,
	runID domain.RunID,
	digest policy.ApprovalDigest,
	terminate bool,
) (Result, error) {
	l.mu.Lock()
	pending, ok := l.pending[runID]
	if !ok || pending.digest != digest {
		ok = false
	}
	l.mu.Unlock()
	if !ok {
		return Result{}, ErrApprovalRejectionMismatch
	}
	if err := l.event(ctx, pending.run, "ApprovalRejected", struct {
		Digest policy.ApprovalDigest `json:"digest"`
	}{Digest: digest}); err != nil {
		return Result{}, err
	}
	if terminate {
		result, err := l.stop(ctx, &pending.run, "APPROVAL_REJECTED")
		if err == nil {
			l.deletePending(runID, digest)
		}
		return result, err
	}
	rejected := l.feedback(
		pending.run,
		"APPROVAL_REJECTED",
		domain.Observation{Stdout: "user rejected the pending action"},
	)
	if err := l.event(ctx, pending.run, "FeedbackProduced", rejected); err != nil {
		return Result{}, err
	}
	if err := l.transition(ctx, &pending.run, domain.StateDeciding, "ApprovalRejectedContinued"); err != nil {
		return Result{}, err
	}
	l.deletePending(runID, digest)
	return l.run(ctx, pending.run, rejected)
}
func (l *Loop) await(ctx context.Context, run *domain.Run, request ApprovalRequired) (Result, error) {
	if err := l.transitionPayload(ctx, run, domain.StateAwaitingApproval, "ApprovalRequired", request); err != nil {
		return Result{}, err
	}
	return Result{State: run.State, StopCode: "APPROVAL_REQUIRED"}, nil
}
func (l *Loop) stop(ctx context.Context, run *domain.Run, code string) (Result, error) {
	if run.State != domain.StateStopped {
		if err := domain.Transition(run.State, domain.StateStopped); err != nil {
			return Result{}, err
		}
		expected := run.State
		stopped := *run
		stopped.State = domain.StateStopped
		payload, _ := json.Marshal(struct {
			Reason string `json:"reason"`
		}{Reason: code})
		if _, err := l.d.Store.UpdateRunIfState(
			context.WithoutCancel(ctx), stopped, expected, "RunStopped", payload,
		); err != nil {
			return Result{}, err
		}
		*run = stopped
	}
	return Result{State: run.State, StopCode: code}, nil
}
func (l *Loop) transition(ctx context.Context, run *domain.Run, to domain.RunState, event string) error {
	return l.transitionPayload(ctx, run, to, event, json.RawMessage(`{}`))
}
func (l *Loop) transitionPayload(
	ctx context.Context,
	run *domain.Run,
	to domain.RunState,
	event string,
	value any,
) error {
	if err := domain.Transition(run.State, to); err != nil {
		return err
	}
	expected := run.State
	updated := *run
	updated.State = to
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := l.d.Store.UpdateRunIfState(ctx, updated, expected, event, payload); err != nil {
		return err
	}
	*run = updated
	return nil
}
func (l *Loop) event(ctx context.Context, run domain.Run, typ string, payload any) error {
	data, _ := json.Marshal(payload)
	_, err := l.d.Store.AppendEvent(ctx, run.ID, typ, data)
	return err
}
func (l *Loop) feedback(run domain.Run, code string, obs domain.Observation) *domain.StructuredFeedback {
	v := l.d.Feedback.Process(feedback.Input{StageID: run.CurrentStage, Code: code, Observation: obs})
	return &v
}
func (l *Loop) deletePending(runID domain.RunID, digest policy.ApprovalDigest) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if pending, ok := l.pending[runID]; ok && pending.digest == digest {
		delete(l.pending, runID)
	}
}
func valid(d domain.AgentDecision) bool {
	return d.Version == "1" && d.Action.Kind != "" && json.Valid(d.Action.Args)
}
