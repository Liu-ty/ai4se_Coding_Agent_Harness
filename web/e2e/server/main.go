// Command server is a test-only production-equivalent HTTP boundary for Playwright.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
)

const (
	localAddress = "127.0.0.1:4174"
	demoAddress  = "127.0.0.1:4175"
)

type deterministicReader struct {
	mu    sync.Mutex
	value byte
}

func (r *deterministicReader) Read(buffer []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value++
	for index := range buffer {
		buffer[index] = r.value
	}
	return len(buffer), nil
}

type fixtureApplication struct {
	store *store.MemoryStore
	mu    sync.Mutex
	next  int
}

func (a *fixtureApplication) CreateRun(
	ctx context.Context,
	request app.CreateRunRequest,
) (domain.Run, error) {
	a.mu.Lock()
	a.next++
	runID := domain.RunID(fmt.Sprintf("run-created-%d", a.next))
	a.mu.Unlock()
	now := time.Now().UTC()
	run := domain.Run{
		ID: runID, State: domain.StateAwaitingApproval, Profile: request.Profile,
		Task: request.Task, RepoRoot: request.RepoRoot, CurrentStage: "targeted-test",
		CreatedAt: now, UpdatedAt: now,
	}
	payload := json.RawMessage(`{"summary":"Run created","budgets":{"decisions":{"used":1,"limit":30},"mutations":{"used":0,"limit":5}}}`)
	if _, err := a.store.CreateRunWithEvent(ctx, run, "RunCreated", payload); err != nil {
		return domain.Run{}, err
	}
	artifactID := "diff-" + string(runID)
	approval, _ := json.Marshal(map[string]any{
		"summary": "Exact patch requires approval", "digest": "digest-" + string(runID),
		"action": "apply_patch", "files": []string{"src/sum.ts"},
		"risk": "Modifies one tracked source file", "artifact_id": artifactID,
		"budgets": map[string]any{
			"decisions": map[string]int{"used": 2, "limit": 30},
			"mutations": map[string]int{"used": 1, "limit": 5},
		},
	})
	if _, err := a.store.AppendEvent(ctx, runID, "ApprovalRequired", approval); err != nil {
		return domain.Run{}, err
	}
	err := a.store.PutArtifact(ctx, domain.Artifact{
		ID: artifactID, RunID: runID, Kind: "diff",
		Content: []byte("--- a/src/sum.ts\n+++ b/src/sum.ts\n- return 1\n+ return a + b\n"),
	})
	return run, err
}

func (a *fixtureApplication) GetRun(ctx context.Context, runID domain.RunID) (domain.Run, error) {
	return a.store.GetRun(ctx, runID)
}

func (a *fixtureApplication) CancelRun(ctx context.Context, runID domain.RunID) error {
	run, err := a.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	run.State = domain.StateStopped
	run.UpdatedAt = time.Now().UTC()
	_, err = a.store.UpdateRun(ctx, run, "RunStopped", json.RawMessage(
		`{"summary":"User cancelled the run","reason":"USER_CANCELLED"}`,
	))
	return err
}

func (a *fixtureApplication) Approve(ctx context.Context, runID domain.RunID, digest string) error {
	if digest != "digest-"+string(runID) {
		return app.ErrApprovalStale
	}
	run, err := a.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	run.State = domain.StateValidating
	run.UpdatedAt = time.Now().UTC()
	_, err = a.store.UpdateRun(ctx, run, "ApprovalAccepted", json.RawMessage(
		`{"summary":"Exact action approved once","budgets":{"decisions":{"used":2,"limit":30},"mutations":{"used":1,"limit":5}}}`,
	))
	return err
}

func (a *fixtureApplication) Reject(
	ctx context.Context,
	runID domain.RunID,
	digest string,
	terminate bool,
) error {
	if digest != "digest-"+string(runID) {
		return app.ErrApprovalStale
	}
	run, err := a.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if terminate {
		run.State = domain.StateStopped
	} else {
		run.State = domain.StateDeciding
	}
	run.UpdatedAt = time.Now().UTC()
	_, err = a.store.UpdateRun(ctx, run, "ApprovalRejected", json.RawMessage(
		`{"summary":"Exact action rejected","reason":"APPROVAL_REJECTED"}`,
	))
	return err
}

func (*fixtureApplication) Preflight(
	_ context.Context,
	request app.CreateRunRequest,
) app.PreflightReport {
	return app.PreflightReport{
		OK: true, RepoRoot: request.RepoRoot, BaselineCommit: "fixture-commit",
		BaselineDiffHash: "fixture-diff",
		Findings: []app.Finding{{
			Code: "REPOSITORY_REACHABLE", Severity: app.SeverityInfo,
			Message: "Repository reachable",
		}},
	}
}

type fixtureCredentials struct {
	mu         sync.Mutex
	configured bool
}

func (c *fixtureCredentials) Add(_ context.Context, _ credentials.Ref, secret []byte) error {
	if len(secret) == 0 {
		return credentials.ErrInvalidCredential
	}
	c.mu.Lock()
	c.configured = true
	c.mu.Unlock()
	return nil
}

func (c *fixtureCredentials) Update(ctx context.Context, ref credentials.Ref, secret []byte) error {
	return c.Add(ctx, ref, secret)
}

func (c *fixtureCredentials) Status(_ context.Context, ref credentials.Ref) (credentials.Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return credentials.Status{
		Ref: ref, Configured: c.configured, Backend: "memory-fixture",
		UpdatedAt: time.Date(2026, 7, 29, 1, 2, 3, 0, time.UTC),
	}, nil
}

func (c *fixtureCredentials) Clear(_ context.Context, _ credentials.Ref) error {
	c.mu.Lock()
	c.configured = false
	c.mu.Unlock()
	return nil
}

func main() {
	dist, err := filepath.Abs(filepath.Join("..", "internal", "httpapi", "webdist"))
	if err != nil {
		panic(err)
	}
	index, err := os.ReadFile(filepath.Join(dist, "index.html"))
	if err != nil {
		panic("build web assets before E2E: " + err.Error())
	}

	localStore := store.NewMemory()
	localApp := &fixtureApplication{store: localStore}
	seedLocal(localStore)
	random := &deterministicReader{}
	var localRouter *httpapi.Router
	localShell := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		runtime := runtimeScript(localRouter.CSRFToken(), false, nil)
		serveIndex(writer, index, runtime)
	})
	localRouter, err = httpapi.NewLocal(httpapi.Options{
		Application: localApp, Store: localStore, Credentials: &fixtureCredentials{},
		Capabilities: httpapi.LocalCapabilities(), Host: localAddress, Random: random,
		AppShell: localShell, PollInterval: 10 * time.Millisecond,
		HeartbeatInterval: 100 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}

	demoStore := store.NewMemory()
	demoApp := &fixtureApplication{store: demoStore}
	seedDemo(demoStore)
	demoRouter, err := httpapi.NewDemo(httpapi.Options{
		Application: demoApp, Store: demoStore,
		Capabilities: httpapi.DemoCapabilities("demo-feedback"),
		PollInterval: 10 * time.Millisecond, HeartbeatInterval: 100 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}

	localHandler := assetWrapper(dist, localRouter, nil)
	demoShell := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		serveIndex(writer, index, runtimeScript("", true, []string{"demo-feedback"}))
	})
	demoHandler := assetWrapper(dist, demoRouter, demoShell)
	go func() {
		if serveErr := http.ListenAndServe(localAddress, localHandler); serveErr != nil &&
			!errors.Is(serveErr, http.ErrServerClosed) {
			panic(serveErr)
		}
	}()
	if serveErr := http.ListenAndServe(demoAddress, demoHandler); serveErr != nil {
		panic(serveErr)
	}
}

func seedLocal(target *store.MemoryStore) {
	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	run := domain.Run{
		ID: "run-seeded", State: domain.StateAwaitingApproval, Profile: domain.ProfileSupervised,
		Task: "Repair seeded check", RepoRoot: `C:\fixture`, CurrentStage: "targeted-test",
		CreatedAt: now, UpdatedAt: now,
	}
	_, _ = target.CreateRunWithEvent(context.Background(), run, "RunCreated",
		json.RawMessage(`{"summary":"Seeded run","budgets":{"decisions":{"used":1,"limit":30},"mutations":{"used":0,"limit":5}}}`))
}

func seedDemo(target *store.MemoryStore) {
	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	run := domain.Run{
		ID: "demo-feedback", State: domain.StateSucceeded, Profile: domain.ProfileWorkspaceAuto,
		Task: "SIMULATED feedback-loop repair", RepoRoot: "SIMULATED/workspace",
		CurrentStage: "final", CreatedAt: now, UpdatedAt: now,
	}
	_, _ = target.CreateRunWithEvent(context.Background(), run, "PolicyEvaluated",
		json.RawMessage(`{"simulated":true,"category":"POLICY_DENIED","summary":"Guardrail intercepted a protected path."}`))
	_, _ = target.AppendEvent(context.Background(), run.ID, "ValidationFailed",
		json.RawMessage(`{"simulated":true,"category":"TEST_FAILURE","summary":"Injected validation failure.","budgets":{"decisions":{"used":2,"limit":30},"mutations":{"used":1,"limit":5}}}`))
	_, _ = target.AppendEvent(context.Background(), run.ID, "DecisionChanged",
		json.RawMessage(`{"simulated":true,"category":"ACTION_CHANGED","summary":"Feedback changed the second patch."}`))
	_, _ = target.AppendEvent(context.Background(), run.ID, "RunSucceeded",
		json.RawMessage(`{"simulated":true,"category":"SUCCEEDED","summary":"All required checks passed.","reason":"ALL_REQUIRED_CHECKS_PASSED","diff":"--- a/sum.ts\n+++ b/sum.ts\n- return 1\n+ return a + b\n"}`))
}

func assetWrapper(dist string, api http.Handler, shell http.Handler) http.Handler {
	files := http.FileServer(http.Dir(dist))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/healthz" {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.HasPrefix(request.URL.Path, "/assets/") {
			files.ServeHTTP(writer, request)
			return
		}
		if shell != nil && request.URL.Path == "/" {
			shell.ServeHTTP(writer, request)
			return
		}
		api.ServeHTTP(writer, request)
	})
}

func serveIndex(writer http.ResponseWriter, index []byte, runtime string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write([]byte(strings.Replace(
		string(index), "</head>", runtime+"</head>", 1,
	)))
}

func runtimeScript(csrf string, demo bool, fixedRuns []string) string {
	encoded, _ := json.Marshal(map[string]any{
		"csrfToken": csrf,
		"capabilities": map[string]any{
			"createRuns": !demo, "cancelRuns": !demo, "approvals": !demo,
			"artifacts": !demo, "configValidation": !demo, "credentials": !demo,
			"demo": demo, "fixedRuns": fixedRuns,
		},
	})
	return `<script>window.__AI4SE_RUNTIME__=` + string(encoded) + `;</script>`
}
