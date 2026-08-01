package demo

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/budget"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
)

// Composition is a fully in-memory public-demo boundary. It has no host
// filesystem, credential store, custom endpoint, or process executor.
type Composition struct {
	store     *store.MemoryStore
	router    *httpapi.Router
	workspace map[string]string
	result    Result
}

func NewComposition(ctx context.Context, _ string) (*Composition, error) {
	composition := &Composition{store: store.NewMemory(), workspace: make(map[string]string)}
	if err := composition.run(ctx); err != nil {
		return nil, err
	}
	router, err := httpapi.NewDemo(httpapi.Options{
		Application:  composition,
		Store:        composition.store,
		Capabilities: httpapi.DemoCapabilities(feedbackLoopRunID),
		AppShell:     httpapi.WebHandler(),
	})
	if err != nil {
		return nil, err
	}
	composition.router = router
	return composition, nil
}

func (c *Composition) run(ctx context.Context) error {
	run := domain.Run{ID: feedbackLoopRunID, State: domain.StateCreated, Profile: domain.ProfileWorkspaceAuto, Task: "repair the deterministic demo", RepoRoot: "in-memory"}
	if err := c.store.CreateRun(ctx, run); err != nil {
		return err
	}
	provider := newScriptedProvider()
	loop := agent.New(agent.Dependencies{
		Store: c.store, Provider: provider, Actions: inMemoryExecutor{store: c},
		Policy: policy.NewEngine(), Feedback: feedback.Pipeline{}, Validation: &deterministicValidator{},
		Budget: budget.New(budget.Limits{MaxDecisions: 4, MaxMutations: 2, MaxProtocolRepairs: 1, WallClock: time.Hour}, wallClock{}),
	})
	result, err := loop.Run(ctx, run)
	if err != nil {
		return err
	}
	events, err := c.store.ListEvents(ctx, feedbackLoopRunID, 0)
	if err != nil {
		return err
	}
	mechanism := make([]string, 0, 7)
	for _, event := range events {
		switch event.Type {
		case "PolicyDenied", "PatchApplied", "ValidationFailed", "ValidationPassed", "RunSucceeded":
			mechanism = append(mechanism, event.Type)
		case "FeedbackProduced":
			if len(mechanism) > 0 && mechanism[len(mechanism)-1] == "ValidationFailed" {
				mechanism = append(mechanism, event.Type)
			}
		}
	}
	c.result = Result{State: result.State, Actions: provider.RecordedActions(), Events: mechanism}
	return nil
}

func (c *Composition) Result() Result       { return c.result }
func (c *Composition) Router() http.Handler { return c.router }

// The following app.Application methods deliberately make every mutable local
// capability unreachable from the demo router.
func (*Composition) CreateRun(context.Context, app.CreateRunRequest) (domain.Run, error) {
	return domain.Run{}, errors.New("demo: run creation is disabled")
}
func (c *Composition) GetRun(ctx context.Context, id domain.RunID) (domain.Run, error) {
	return c.store.GetRun(ctx, id)
}
func (*Composition) CancelRun(context.Context, domain.RunID) error {
	return errors.New("demo: cancellation is disabled")
}
func (*Composition) Approve(context.Context, domain.RunID, string) error {
	return errors.New("demo: approvals are disabled")
}
func (*Composition) Reject(context.Context, domain.RunID, string, bool) error {
	return errors.New("demo: approvals are disabled")
}
func (*Composition) Preflight(context.Context, app.CreateRunRequest) app.PreflightReport {
	return app.PreflightReport{}
}

var _ httpapi.Application = (*Composition)(nil)
