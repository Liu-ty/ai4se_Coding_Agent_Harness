// Package demo provides a deterministic, mock-only feedback-loop demonstration.
package demo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
)

const feedbackLoopRunID domain.RunID = "demo-feedback-loop"

// Action records a proposed course action without retaining any host capability.
type Action struct {
	Kind   string `json:"kind"`
	Digest string `json:"digest"`
}

// Result is the stable public output of the deterministic demo.
type Result struct {
	State   domain.RunState `json:"state"`
	Actions []Action        `json:"actions"`
	Events  []string        `json:"events"`
}

func (r Result) EventTypes() []string {
	return append([]string(nil), r.Events...)
}

// RunFeedbackLoop runs the fixed policy-denial, repair, and validation demo.
func RunFeedbackLoop(ctx context.Context) (Result, error) {
	composition, err := NewComposition(ctx, "")
	if err != nil {
		return Result{}, err
	}
	return composition.Result(), nil
}

type scriptedProvider struct {
	mu      sync.Mutex
	index   int
	actions []domain.Action
}

func newScriptedProvider() *scriptedProvider {
	return &scriptedProvider{actions: []domain.Action{
		{Kind: "raw_shell", Args: json.RawMessage(`{}`)},
		{Kind: "apply_patch", Args: json.RawMessage(`{"patch":"--- a/demo.txt\n+++ b/demo.txt\n@@ -1 +1 @@\n-broken\n+almost fixed\n"}`)},
		{Kind: "apply_patch", Args: json.RawMessage(`{"patch":"--- a/demo.txt\n+++ b/demo.txt\n@@ -1 +1 @@\n-almost fixed\n+fixed\n"}`)},
		{Kind: "finish", Args: json.RawMessage(`{}`)},
	}}
}

func (p *scriptedProvider) Decide(_ context.Context, request provider.Request) (provider.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.index == 1 && (request.LastFeedback == nil || request.LastFeedback.Category != "POLICY_DENIED") {
		return provider.Response{}, errors.New("demo: policy feedback was not supplied")
	}
	if p.index == 2 && (request.LastFeedback == nil || request.LastFeedback.Category != "TEST_FAILURE") {
		return provider.Response{}, errors.New("demo: test feedback was not supplied")
	}
	if p.index >= len(p.actions) {
		return provider.Response{}, errors.New("demo: script exhausted")
	}
	action := p.actions[p.index]
	p.index++
	return provider.Response{Decision: domain.AgentDecision{Version: "1", Action: action}}, nil
}

func (p *scriptedProvider) RecordedActions() []Action {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]Action, p.index)
	for index, action := range p.actions[:p.index] {
		sum := sha256.Sum256(append([]byte(action.Kind+":"), action.Args...))
		result[index] = Action{Kind: action.Kind, Digest: hex.EncodeToString(sum[:])}
	}
	return result
}

type inMemoryExecutor struct {
	store *Composition
}

func (e inMemoryExecutor) Execute(ctx context.Context, action domain.Action) (agent.ActionResult, error) {
	if action.Kind != "apply_patch" {
		return agent.ActionResult{}, fmt.Errorf("demo: unsupported executable action %q", action.Kind)
	}
	var input struct {
		Patch string `json:"patch"`
	}
	if err := json.Unmarshal(action.Args, &input); err != nil || input.Patch == "" {
		return agent.ActionResult{}, errors.New("demo: invalid in-memory patch")
	}
	sum := sha256.Sum256([]byte(input.Patch))
	digest := hex.EncodeToString(sum[:])
	e.store.workspace["demo.txt"] = digest
	if _, err := e.store.store.AppendEvent(ctx, feedbackLoopRunID, "PatchApplied", json.RawMessage(`{"workspace":"in-memory"}`)); err != nil {
		return agent.ActionResult{}, err
	}
	return agent.ActionResult{DiffDigest: digest}, nil
}

type deterministicValidator struct{ current int }

func (v *deterministicValidator) Baseline(context.Context, domain.Run) agent.ValidationResult {
	return agent.ValidationResult{StageID: "demo-test", Passed: false, Observation: failedObservation()}
}

func (v *deterministicValidator) Current(context.Context, domain.Run) agent.ValidationResult {
	v.current++
	if v.current == 1 {
		return agent.ValidationResult{StageID: "demo-test", Passed: false, Observation: failedObservation()}
	}
	return agent.ValidationResult{StageID: "demo-test", Passed: true, Observation: passedObservation()}
}

func (v *deterministicValidator) Final(context.Context, domain.Run) agent.ValidationResult {
	return agent.ValidationResult{StageID: "demo-test", Passed: true, Observation: passedObservation()}
}

func failedObservation() domain.Observation {
	code := 1
	return domain.Observation{Code: "EXIT", ExitCode: &code, Stdout: "--- FAIL: TestDemo\nexpected corrected patch"}
}

func passedObservation() domain.Observation {
	code := 0
	return domain.Observation{Code: "EXIT", ExitCode: &code, Stdout: "PASS"}
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }
