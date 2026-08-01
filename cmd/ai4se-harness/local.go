package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/budget"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/executor"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/tools"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/validation"
)

type localRuntimeOptions struct {
	dataDir     string
	credentials *credentials.Service
}

type localRuntime struct {
	Application *app.Service
	Store       *store.SQLiteStore
	Credentials *credentials.Service
}

func (r *localRuntime) Close() error { return r.Store.Close() }

func newLocalRuntime(ctx context.Context, repo string, options localRuntimeOptions) (*localRuntime, error) {
	dataDir := options.dataDir
	if dataDir == "" {
		dataDir = filepath.Join(repo, ".ai4se-harness")
	}
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create local runtime directory: %w", err)
	}
	storage, err := store.OpenSQLite(filepath.Join(dataDir, "runs.db"))
	if err != nil {
		return nil, err
	}
	creds := options.credentials
	if creds == nil {
		creds = credentials.NewService(credentials.NewKeyringStore(), nil)
	}
	redactor := feedback.NewRedactor(nil)
	factory := app.AgentLoopFactory(func(_ context.Context, setup app.RunSetup) (*agent.Loop, *policy.ApprovalStore, error) {
		checks, err := resolvedChecks(setup.Config)
		if err != nil {
			return nil, nil, err
		}
		decisionProvider, err := localProvider(setup.Request, creds)
		if err != nil {
			return nil, nil, err
		}
		runner := executor.NewLocal()
		registry, err := tools.NewRegistry(
			tools.NewListTool(setup.Report.RepoRoot, 1000), tools.NewSearchTool(setup.Report.RepoRoot, 1000),
			tools.NewReadTool(setup.Report.RepoRoot, 256*1024),
			tools.NewPatchTool(setup.Report.RepoRoot, tools.PatchLimits{}), tools.NewCreateTool(setup.Report.RepoRoot, 1<<20),
		)
		if err != nil {
			return nil, nil, err
		}
		pipeline := validation.New(checks, runner)
		approvals := policy.NewApprovalStore()
		return agent.New(agent.Dependencies{
			Store: storage, Provider: decisionProvider, Actions: localActions{registry: registry, checks: checks, runner: runner},
			Policy: policy.NewEngine(), Feedback: feedback.Pipeline{}, Validation: localValidator{pipeline: pipeline},
			Approvals: approvals, Budget: budget.New(budget.Limits{MaxDecisions: setup.Config.Budget.MaxDecisions, MaxMutations: setup.Config.Budget.MaxMutations, MaxProtocolRepairs: setup.Config.Budget.MaxProtocolRepairs, WallClock: parseWallClock(setup.Config.Budget.WallClock)}, timeClock{}),
		}), approvals, nil
	})
	service, err := app.NewLocal(ctx, storage, factory, creds, dataDir, &redactor)
	if err != nil {
		_ = storage.Close()
		return nil, err
	}
	return &localRuntime{Application: service, Store: storage, Credentials: creds}, nil
}

func localRunRequest(repo, task, configPath string) app.CreateRunRequest {
	return app.CreateRunRequest{RepoRoot: repo, Task: task, ConfigPath: configPath, Provider: "openai", Model: "gpt-4o-mini", Endpoint: "https://api.openai.com", Profile: domain.ProfileWorkspaceAuto}
}

type timeClock struct{}

func (timeClock) Now() time.Time { return time.Now() }

func parseWallClock(raw string) time.Duration {
	value, err := time.ParseDuration(raw)
	if err != nil {
		return time.Minute
	}
	return value
}

func resolvedChecks(cfg config.Config) ([]config.CommandSpec, error) {
	checks := make([]config.CommandSpec, 0, len(cfg.Validation))
	for _, stage := range cfg.Validation {
		check, err := config.ResolveStage(stage, runtime.GOOS)
		if err != nil {
			return nil, err
		}
		checks = append(checks, check)
	}
	return checks, nil
}

func localProvider(request app.CreateRunRequest, creds *credentials.Service) (provider.Provider, error) {
	options := provider.Options{Model: request.Model, ConfirmCustomEndpoint: request.ConfirmCustomEndpoint}
	switch request.Provider {
	case "openai":
		return provider.NewOpenAI(request.Endpoint, nil, creds, options)
	case "anthropic":
		return provider.NewAnthropic(request.Endpoint, nil, creds, options)
	default:
		return nil, fmt.Errorf("unsupported provider %q", request.Provider)
	}
}

type localActions struct {
	registry *tools.Registry
	checks   []config.CommandSpec
	runner   executor.Executor
}

func (a localActions) Execute(ctx context.Context, action domain.Action) (agent.ActionResult, error) {
	if action.Kind == "run_check" {
		var input struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(action.Args, &input); err != nil {
			return agent.ActionResult{}, err
		}
		for _, check := range a.checks {
			if check.ID == input.ID {
				observation, err := a.runner.Run(ctx, check)
				return agent.ActionResult{Observation: observation}, err
			}
		}
		return agent.ActionResult{}, errors.New("configured check not found")
	}
	result, err := a.registry.Execute(ctx, action)
	return agent.ActionResult{DiffDigest: result.SHA256, Observation: domain.Observation{Stdout: result.Text}}, err
}

type localValidator struct{ pipeline *validation.Pipeline }

func (v localValidator) Baseline(ctx context.Context, _ domain.Run) agent.ValidationResult {
	return validationResult(v.pipeline.RunAllRequired(ctx))
}
func (v localValidator) Current(ctx context.Context, _ domain.Run) agent.ValidationResult {
	return validationResult(v.pipeline.RunFrom(ctx, 0))
}
func (v localValidator) Final(ctx context.Context, _ domain.Run) agent.ValidationResult {
	return validationResult(v.pipeline.RunAllRequired(ctx))
}
func validationResult(result validation.Result) agent.ValidationResult {
	if len(result.Stages) == 0 {
		return agent.ValidationResult{StageID: "validation", Passed: false}
	}
	stage := result.Stages[len(result.Stages)-1]
	return agent.ValidationResult{StageID: stage.Stage.ID, Passed: result.Complete, Observation: stage.Observation}
}
