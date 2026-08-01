package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/budget"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/executor"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

// The protected production break is replacing conditional feedback-driven
// decisions with a fixed action sequence.
func TestLoopUsesFailureToChangeNextPatch(t *testing.T) {
	run := newRun()
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	first := action("raw_shell", `{}`)
	broken := patchAction("- return 1", "+ return 0")
	correct := patchAction("- return 1", "+ return 2")
	mock := provider.NewMock(func(_ context.Context, request provider.Request) (provider.Response, error) {
		switch {
		case request.LastFeedback == nil:
			return provider.Response{Decision: decision(first)}, nil
		case request.LastFeedback.Category == "POLICY_DENIED":
			return provider.Response{Decision: decision(broken)}, nil
		case request.LastFeedback.Category == "TEST_FAILURE":
			return provider.Response{Decision: decision(correct)}, nil
		default:
			return provider.Response{Decision: decision(action("finish", `{}`))}, nil
		}
	})
	checks := &checks{results: []agent.ValidationResult{fail("unit"), pass("unit"), pass("full")}}
	loop := agent.New(agent.Dependencies{
		Store: mem, Provider: mock, Actions: runner{}, Policy: policy.NewEngine(),
		Feedback: feedback.Pipeline{}, Validation: checks,
		Budget: budget.New(budget.Limits{MaxDecisions: 8, MaxMutations: 4, MaxProtocolRepairs: 2, WallClock: time.Minute}, fixedClock{}),
	})

	got, err := loop.Run(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateSucceeded {
		t.Fatalf("state = %s, want %s", got.State, domain.StateSucceeded)
	}
	calls := mock.Calls()
	if len(calls) != 4 || calls[2].Request.LastFeedback == nil || calls[2].Request.LastFeedback.Fingerprint == "" {
		t.Fatalf("corrective request lacks validation feedback: %#v", calls)
	}
	if string(calls[1].Returned.Decision.Action.Args) == string(calls[2].Returned.Decision.Action.Args) {
		t.Fatal("patch did not change after test feedback")
	}
	assertEvents(t, mem, run.ID, "PolicyDenied", "ValidationFailed", "FeedbackProduced", "ValidationPassed", "RunSucceeded")
}

func TestLoopStopsAfterProtocolRepairBudget(t *testing.T) {
	run := newRun()
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	loop := agent.New(agent.Dependencies{Store: mem, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
		return provider.Response{}, errors.New("INVALID_JSON")
	}), Actions: runner{}, Policy: policy.NewEngine(), Feedback: feedback.Pipeline{}, Validation: &checks{}, Budget: budget.New(budget.Limits{MaxDecisions: 4, MaxMutations: 1, MaxProtocolRepairs: 2, WallClock: time.Minute}, fixedClock{})})
	got, err := loop.Run(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if got.StopCode != "PROTOCOL_EXHAUSTED" {
		t.Fatalf("stop code = %q", got.StopCode)
	}
	assertEvents(t, mem, run.ID, "ProtocolError", "ProtocolError", "RunStopped")
}

// This protects the one-use approval boundary: a grant must resume only the
// exact stored action, and a consumed grant must not execute it again.
func TestResumeApprovalConsumesExactPendingActionOnce(t *testing.T) {
	run := newRun()
	run.Profile = domain.ProfileSupervised
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	patch := patchAction("- return 1", "+ return 2")
	var decisions int
	mock := provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
		decisions++
		if decisions == 1 {
			return provider.Response{Decision: decision(patch)}, nil
		}
		return provider.Response{Decision: decision(action("finish", `{}`))}, nil
	})
	approvals := policy.NewApprovalStore()
	exec := &countingRunner{}
	loop := agent.New(agent.Dependencies{Store: mem, Provider: mock, Actions: exec, Policy: policy.NewEngine(), Approvals: approvals, Feedback: feedback.Pipeline{}, Validation: &checks{results: []agent.ValidationResult{pass("unit"), pass("full")}}, Budget: budget.New(budget.Limits{MaxDecisions: 4, MaxMutations: 2, MaxProtocolRepairs: 2, WallClock: time.Minute}, fixedClock{})})

	pending, err := loop.Run(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != domain.StateAwaitingApproval || exec.calls != 0 {
		t.Fatalf("pending=%#v calls=%d", pending, exec.calls)
	}
	digest := policy.Digest(run.ID, domain.ProfileSupervised, patch, nil)
	if _, err := loop.ResumeApproval(context.Background(), run.ID, digest); err == nil || exec.calls != 0 {
		t.Fatal("ungranted approval executed action")
	}
	approvals.Grant(digest)
	resumed, err := loop.ResumeApproval(context.Background(), run.ID, digest)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State != domain.StateSucceeded || exec.calls != 1 {
		t.Fatalf("resumed=%#v calls=%d", resumed, exec.calls)
	}
	if _, err := loop.ResumeApproval(context.Background(), run.ID, digest); err == nil || exec.calls != 1 {
		t.Fatal("consumed approval executed action twice")
	}
	assertEvents(t, mem, run.ID, "ApprovalRequired", "ApprovalGranted", "RunSucceeded")
}

func TestApprovalDigestBindsRunBaselines(t *testing.T) {
	run := newRun()
	run.Profile = domain.ProfileSupervised
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	patch := patchAction("- return 1", "+ return 2")
	approvals := policy.NewApprovalStore()
	loop := agent.New(agent.Dependencies{
		Store: mem, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
			return provider.Response{Decision: decision(patch)}, nil
		}), Actions: &countingRunner{}, Policy: policy.NewEngine(), Approvals: approvals,
		Baselines: map[string]string{"worktree": "current-baseline"}, Feedback: feedback.Pipeline{},
		Validation: &checks{}, Budget: budget.New(budget.Limits{MaxDecisions: 2, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute}, fixedClock{}),
	})
	if _, err := loop.Run(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	stale := policy.Digest(run.ID, run.Profile, patch, map[string]string{"worktree": "stale-baseline"})
	approvals.Grant(stale)
	if _, err := loop.ResumeApproval(context.Background(), run.ID, stale); !errors.Is(err, agent.ErrApprovalNotGranted) {
		t.Fatalf("stale baseline approval error = %v", err)
	}
}

func TestApprovalRequiredPublishesRedactedBoundProductionRequest(t *testing.T) {
	const canary = "correct-horse-7429"
	run := newRun()
	run.Profile = domain.ProfileSupervised
	directory := t.TempDir()
	databasePath := filepath.Join(directory, "runs.db")
	db, err := store.OpenSQLite(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = db.Close()
		}
	}()
	if err := db.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	patchText := "--- a/bug.go\n+++ b/bug.go\n@@ -1 +1 @@ " + canary + "\n-old\n+" +
		strings.Repeat("x", 12<<10) +
		"\n--- a/obsolete.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-delete me\n"
	patchArgs, err := json.Marshal(map[string]string{"patch": patchText})
	if err != nil {
		t.Fatal(err)
	}
	patch := domain.Action{
		Kind: "apply_patch",
		Args: patchArgs,
	}
	baselines := map[string]string{"baseline_diff_hash": "diff-2", "baseline_commit": "commit-1"}
	loop := agent.New(agent.Dependencies{
		Store: db, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
			return provider.Response{Decision: decision(patch)}, nil
		}), Actions: &countingRunner{}, Policy: policy.NewEngine(),
		Baselines: baselines, Feedback: feedback.Pipeline{}, Validation: &checks{},
		ApprovalRedactor: feedback.NewRedactor([]string{canary}),
		Budget:           budget.New(budget.Limits{MaxDecisions: 2, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute}, fixedClock{}),
	})

	if _, err := loop.Run(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	events, err := db.ListEvents(context.Background(), run.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	var request agent.ApprovalRequired
	for _, event := range events {
		if event.Type == "ApprovalRequired" {
			if err := json.Unmarshal(event.Payload, &request); err != nil {
				t.Fatal(err)
			}
		}
	}
	if request.Digest != policy.Digest(run.ID, run.Profile, patch, baselines) {
		t.Fatalf("digest = %q", request.Digest)
	}
	var display struct {
		Patch struct {
			SHA256    string `json:"sha256"`
			Preview   string `json:"preview"`
			Truncated bool   `json:"truncated"`
		} `json:"patch"`
	}
	if err := json.Unmarshal(request.Action.Args, &display); err != nil {
		t.Fatal(err)
	}
	if request.Action.Kind != "apply_patch" || display.Patch.SHA256 == "" ||
		!display.Patch.Truncated || !strings.Contains(display.Patch.Preview, "[REDACTED]") ||
		strings.Contains(display.Patch.Preview, "+xxxxxxxx") {
		t.Fatalf("canonical redacted action = %#v", request.Action)
	}
	if len(request.AffectedFiles) != 2 || request.AffectedFiles[0] != "bug.go" ||
		request.AffectedFiles[1] != "obsolete.go" {
		t.Fatalf("affected files = %#v", request.AffectedFiles)
	}
	if request.Risk != policy.RiskNormal || request.RiskReason == "" {
		t.Fatalf("risk = %q reason = %q", request.Risk, request.RiskReason)
	}
	if len(request.BaselineEvidence) != 2 ||
		request.BaselineEvidence[0].Name != "baseline_commit" ||
		request.BaselineEvidence[0].Digest != "commit-1" ||
		request.BaselineEvidence[1].Name != "baseline_diff_hash" {
		t.Fatalf("baseline evidence = %#v", request.BaselineEvidence)
	}
	for _, event := range events {
		if bytes.Contains(event.Payload, []byte(canary)) {
			t.Fatalf("raw event %q leaked a secret: %s", event.Type, event.Payload)
		}
		if event.Type == "ApprovalRequired" && len(event.Payload) > 8<<10 {
			t.Fatalf("approval event is unbounded: %d bytes", len(event.Payload))
		}
	}
	assertSQLiteFilesDoNotContain(t, directory, canary, true)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	assertSQLiteFilesDoNotContain(t, directory, canary, false)
}

func TestApprovedValidationFailureFeedsBackAndRedecides(t *testing.T) {
	run := newRun()
	run.Profile = domain.ProfileSupervised
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	patch := patchAction("- return 1", "+ return 2")
	var sawFeedback bool
	decisions := 0
	mock := provider.NewMock(func(_ context.Context, request provider.Request) (provider.Response, error) {
		decisions++
		if decisions == 1 {
			return provider.Response{Decision: decision(patch)}, nil
		}
		sawFeedback = request.LastFeedback != nil && request.LastFeedback.Category == "TEST_FAILURE"
		return provider.Response{Decision: decision(action("finish", `{}`))}, nil
	})
	approvals := policy.NewApprovalStore()
	loop := agent.New(agent.Dependencies{
		Store: mem, Provider: mock, Actions: &countingRunner{}, Policy: policy.NewEngine(), Approvals: approvals,
		Feedback: feedback.Pipeline{}, Validation: &checks{results: []agent.ValidationResult{fail("unit"), pass("full")}},
		Budget: budget.New(budget.Limits{MaxDecisions: 3, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute}, fixedClock{}),
	})
	if _, err := loop.Run(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	digest := policy.Digest(run.ID, run.Profile, patch, nil)
	approvals.Grant(digest)
	result, err := loop.ResumeApproval(context.Background(), run.ID, digest)
	if err != nil || result.State != domain.StateSucceeded || !sawFeedback {
		t.Fatalf("resumed result/feedback = %#v/%v/%v", result, sawFeedback, err)
	}
	assertEvents(t, mem, run.ID, "ValidationFailed", "FeedbackProduced", "RunSucceeded")
}

func TestRejectApprovalEitherStopsOrFeedsBackWithoutExecution(t *testing.T) {
	t.Run("terminate", func(t *testing.T) {
		run := newRun()
		run.Profile = domain.ProfileSupervised
		mem := store.NewMemory()
		if err := mem.CreateRun(context.Background(), run); err != nil {
			t.Fatal(err)
		}
		patch := patchAction("- return 1", "+ return 2")
		exec := &countingRunner{}
		loop := agent.New(agent.Dependencies{
			Store: mem, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
				return provider.Response{Decision: decision(patch)}, nil
			}), Actions: exec, Policy: policy.NewEngine(), Feedback: feedback.Pipeline{},
			Validation: &checks{}, Budget: budget.New(budget.Limits{
				MaxDecisions: 2, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute,
			}, fixedClock{}),
		})
		pending, err := loop.Run(context.Background(), run)
		if err != nil || pending.State != domain.StateAwaitingApproval {
			t.Fatalf("pending = %#v, %v", pending, err)
		}
		digest := policy.Digest(run.ID, run.Profile, patch, nil)
		stopped, err := loop.RejectApproval(context.Background(), run.ID, digest, true)
		if err != nil {
			t.Fatal(err)
		}
		if stopped.State != domain.StateStopped || stopped.StopCode != "APPROVAL_REJECTED" || exec.calls != 0 {
			t.Fatalf("result/calls = %#v/%d", stopped, exec.calls)
		}
		if _, err := loop.RejectApproval(context.Background(), run.ID, digest, true); err == nil {
			t.Fatal("rejected pending action remained reusable")
		}
	})

	t.Run("feedback", func(t *testing.T) {
		run := newRun()
		run.Profile = domain.ProfileSupervised
		mem := store.NewMemory()
		if err := mem.CreateRun(context.Background(), run); err != nil {
			t.Fatal(err)
		}
		patch := patchAction("- return 1", "+ return 2")
		var sawRejectedFeedback bool
		mock := provider.NewMock(func(_ context.Context, request provider.Request) (provider.Response, error) {
			if request.LastFeedback == nil {
				return provider.Response{Decision: decision(patch)}, nil
			}
			sawRejectedFeedback = request.LastFeedback.Category == "APPROVAL_REJECTED"
			return provider.Response{Decision: decision(action("finish", `{}`))}, nil
		})
		exec := &countingRunner{}
		loop := agent.New(agent.Dependencies{
			Store: mem, Provider: mock, Actions: exec, Policy: policy.NewEngine(),
			Feedback: feedback.Pipeline{}, Validation: &checks{results: []agent.ValidationResult{pass("full")}},
			Budget: budget.New(budget.Limits{
				MaxDecisions: 3, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute,
			}, fixedClock{}),
		})
		if _, err := loop.Run(context.Background(), run); err != nil {
			t.Fatal(err)
		}
		digest := policy.Digest(run.ID, run.Profile, patch, nil)
		result, err := loop.RejectApproval(context.Background(), run.ID, digest, false)
		if err != nil {
			events, _ := mem.ListEvents(context.Background(), run.ID, 1)
			stored, _ := mem.GetRun(context.Background(), run.ID)
			t.Fatalf("%v; stored=%#v events=%#v", err, stored, events)
		}
		if result.State != domain.StateSucceeded || !sawRejectedFeedback || exec.calls != 0 {
			t.Fatalf("result/feedback/calls = %#v/%v/%d", result, sawRejectedFeedback, exec.calls)
		}
		assertEvents(t, mem, run.ID, "ApprovalRejected", "FeedbackProduced", "RunSucceeded")
	})
}

func TestRejectApprovalRetainsPendingActionWhenPersistenceFails(t *testing.T) {
	run := newRun()
	run.Profile = domain.ProfileSupervised
	base := store.NewMemory()
	if err := base.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	st := &failEventStore{Store: base}
	patch := patchAction("- return 1", "+ return 2")
	loop := agent.New(agent.Dependencies{
		Store: st, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
			return provider.Response{Decision: decision(patch)}, nil
		}), Actions: &countingRunner{}, Policy: policy.NewEngine(), Feedback: feedback.Pipeline{},
		Validation: &checks{}, Budget: budget.New(budget.Limits{
			MaxDecisions: 2, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute,
		}, fixedClock{}),
	})
	if _, err := loop.Run(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	digest := policy.Digest(run.ID, run.Profile, patch, nil)
	st.failType = "ApprovalRejected"
	if _, err := loop.RejectApproval(context.Background(), run.ID, digest, true); err == nil {
		t.Fatal("injected persistence failure was not returned")
	}
	st.failType = ""
	result, err := loop.RejectApproval(context.Background(), run.ID, digest, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != domain.StateStopped {
		t.Fatalf("retry result = %#v", result)
	}
}

func TestLoopConditionalTransitionCannotOverwriteNewerTerminalState(t *testing.T) {
	run := newRun()
	run.Profile = domain.ProfileSupervised
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	patch := patchAction("- return 1", "+ return 2")
	approvals := policy.NewApprovalStore()
	loop := agent.New(agent.Dependencies{
		Store: mem, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
			return provider.Response{Decision: decision(patch)}, nil
		}), Actions: &countingRunner{}, Policy: policy.NewEngine(), Approvals: approvals,
		Feedback: feedback.Pipeline{}, Validation: &checks{}, Budget: budget.New(budget.Limits{
			MaxDecisions: 2, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute,
		}, fixedClock{}),
	})
	if _, err := loop.Run(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	stored, err := mem.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.State = domain.StateStopped
	if _, err := mem.UpdateRun(context.Background(), stored, "ConcurrentStop", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	digest := policy.Digest(run.ID, run.Profile, patch, nil)
	approvals.Grant(digest)
	if _, err := loop.ResumeApproval(context.Background(), run.ID, digest); !errors.Is(err, storeport.ErrRunStateChanged) {
		t.Fatalf("resume error = %v, want ErrRunStateChanged", err)
	}
	got, err := mem.GetRun(context.Background(), run.ID)
	if err != nil || got.State != domain.StateStopped {
		t.Fatalf("stored run = %#v, %v", got, err)
	}
}

func TestLoopReturnsToDecidingWhenFinalValidationRegresses(t *testing.T) {
	run := newRun()
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	mock := provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
		return provider.Response{Decision: decision(action("finish", `{}`))}, nil
	})
	loop := agent.New(agent.Dependencies{Store: mem, Provider: mock, Actions: runner{}, Policy: policy.NewEngine(), Feedback: feedback.Pipeline{}, Validation: &checks{results: []agent.ValidationResult{fail("full"), pass("full")}}, Budget: budget.New(budget.Limits{MaxDecisions: 4, MaxMutations: 1, MaxProtocolRepairs: 2, WallClock: time.Minute}, fixedClock{})})
	got, err := loop.Run(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateSucceeded || len(mock.Calls()) != 2 {
		t.Fatalf("result=%#v calls=%d", got, len(mock.Calls()))
	}
	assertEvents(t, mem, run.ID, "ValidationFailed", "FinalRegression", "RunSucceeded")
}

func TestLoopPersistsCancellationAsTerminalState(t *testing.T) {
	run := newRun()
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	loop := agent.New(agent.Dependencies{Store: mem, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
		t.Fatal("provider called after cancellation")
		return provider.Response{}, nil
	}), Actions: runner{}, Policy: policy.NewEngine(), Feedback: feedback.Pipeline{}, Validation: &checks{}, Budget: budget.New(budget.Limits{MaxDecisions: 1, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute}, fixedClock{})})
	got, err := loop.Run(ctx, run)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateStopped || got.StopCode != "USER_CANCELLED" {
		t.Fatalf("result=%#v", got)
	}
	assertEvents(t, mem, run.ID, "RunStopped")
}

func TestLoopStopsOnRepeatedFailureWithoutDiffProgress(t *testing.T) {
	run := newRun()
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	patch := patchAction("- return 1", "+ return 0")
	loop := agent.New(agent.Dependencies{Store: mem, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
		return provider.Response{Decision: decision(patch)}, nil
	}), Actions: runner{}, Policy: policy.NewEngine(), Feedback: feedback.Pipeline{}, Validation: &checks{results: []agent.ValidationResult{fail("unit"), fail("unit")}}, Budget: budget.New(budget.Limits{MaxDecisions: 4, MaxMutations: 4, MaxProtocolRepairs: 2, WallClock: time.Minute}, fixedClock{}), Progress: budget.NewProgressDetector(2)})
	got, err := loop.Run(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if got.StopCode != "NO_PROGRESS" {
		t.Fatalf("result=%#v", got)
	}
	assertEvents(t, mem, run.ID, "ValidationFailed", "RunStopped")
}

func TestReviewFinishCompletesWithoutValidation(t *testing.T) {
	run := newRun()
	run.Profile = domain.ProfileReview
	mem := store.NewMemory()
	if err := mem.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	loop := agent.New(agent.Dependencies{Store: mem, Provider: provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
		return provider.Response{Decision: decision(action("finish", `{}`))}, nil
	}), Actions: runner{}, Policy: policy.NewEngine(), Feedback: feedback.Pipeline{}, Validation: panicChecks{t}, Budget: budget.New(budget.Limits{MaxDecisions: 1, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute}, fixedClock{})})
	got, err := loop.Run(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateReviewComplete {
		t.Fatalf("result=%#v", got)
	}
}

func action(kind, args string) domain.Action {
	return domain.Action{Kind: kind, Args: json.RawMessage(args)}
}
func patchAction(old, new string) domain.Action {
	b, _ := json.Marshal(map[string]string{"patch": "--- a/bug.go\n+++ b/bug.go\n@@ -1 +1 @@\n" + old + "\n" + new})
	return domain.Action{Kind: "apply_patch", Args: b}
}
func decision(a domain.Action) domain.AgentDecision {
	return domain.AgentDecision{Version: "1", Action: a}
}
func newRun() domain.Run {
	return domain.Run{ID: "run-1", State: domain.StatePreflight, Profile: domain.ProfileWorkspaceAuto, Task: "repair"}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Unix(0, 0) }

type runner struct{}

func (runner) Execute(context.Context, domain.Action) (agent.ActionResult, error) {
	return agent.ActionResult{DiffDigest: "changed"}, nil
}

type countingRunner struct{ calls int }

func (r *countingRunner) Execute(context.Context, domain.Action) (agent.ActionResult, error) {
	r.calls++
	return agent.ActionResult{DiffDigest: "changed"}, nil
}

type checks struct{ results []agent.ValidationResult }

type failEventStore struct {
	storeport.Store
	failType string
}

func (s *failEventStore) AppendEvent(
	ctx context.Context,
	id domain.RunID,
	eventType string,
	payload json.RawMessage,
) (domain.RunEvent, error) {
	if eventType == s.failType {
		return domain.RunEvent{}, errors.New("injected event failure")
	}
	return s.Store.AppendEvent(ctx, id, eventType, payload)
}

type panicChecks struct{ t *testing.T }

func (p panicChecks) Baseline(context.Context, domain.Run) agent.ValidationResult {
	p.t.Fatal("baseline called")
	return agent.ValidationResult{}
}
func (p panicChecks) Current(context.Context, domain.Run) agent.ValidationResult {
	p.t.Fatal("current called")
	return agent.ValidationResult{}
}
func (p panicChecks) Final(context.Context, domain.Run) agent.ValidationResult {
	p.t.Fatal("final called")
	return agent.ValidationResult{}
}

func (c *checks) Baseline(context.Context, domain.Run) agent.ValidationResult { return fail("unit") }
func (c *checks) Current(context.Context, domain.Run) agent.ValidationResult {
	r := c.results[0]
	c.results = c.results[1:]
	return r
}
func (c *checks) Final(context.Context, domain.Run) agent.ValidationResult {
	r := c.results[0]
	c.results = c.results[1:]
	return r
}
func fail(stage string) agent.ValidationResult {
	return agent.ValidationResult{StageID: stage, Observation: domain.Observation{Code: executor.CodeExit, ExitCode: intp(1), Stdout: "--- FAIL: TestValue\n expected 2"}}
}
func pass(stage string) agent.ValidationResult {
	return agent.ValidationResult{StageID: stage, Passed: true, Observation: domain.Observation{Code: executor.CodeExit, ExitCode: intp(0)}}
}
func intp(v int) *int { return &v }

func assertSQLiteFilesDoNotContain(
	t *testing.T,
	directory string,
	canary string,
	databaseOpen bool,
) {
	t.Helper()
	phase := "after close"
	if databaseOpen {
		phase = "while open"
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || (entry.Name() != "runs.db" &&
			!strings.HasPrefix(entry.Name(), "runs.db-")) {
			continue
		}
		persisted, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatalf("read SQLite file %q %s: %v", entry.Name(), phase, err)
		}
		if bytes.Contains(persisted, []byte(canary)) {
			t.Fatalf("SQLite file %q contains approval canary %s", entry.Name(), phase)
		}
	}
}

func assertEvents(t *testing.T, mem *store.MemoryStore, id domain.RunID, want ...string) {
	t.Helper()
	events, err := mem.ListEvents(context.Background(), id, 1)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range events {
		got[e.Type] = true
	}
	for _, v := range want {
		if !got[v] {
			t.Fatalf("missing %s in %#v", v, events)
		}
	}
}
