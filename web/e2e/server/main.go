// Command server is a test-only production-equivalent HTTP boundary for Playwright.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/budget"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
)

const (
	localAddress   = "127.0.0.1:4174"
	demoAddress    = "127.0.0.1:4175"
	approvalCanary = "quartz-orchid-7429"
)

type deterministicReader struct {
	mu    sync.Mutex
	value byte
}

type oneShotFailures struct {
	mu     sync.Mutex
	target string
}

func (f *oneShotFailures) arm(target string) bool {
	if target != "runs" && target != "credential" {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.target = target
	return true
}

func (f *oneShotFailures) consume(target string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.target != target {
		return false
	}
	f.target = ""
	return true
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

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type deterministicActions struct{}

func (deterministicActions) Execute(
	context.Context,
	domain.Action,
) (agent.ActionResult, error) {
	return agent.ActionResult{DiffDigest: "e2e-diff"}, nil
}

type deterministicValidation struct{}

func (deterministicValidation) Baseline(
	context.Context,
	domain.Run,
) agent.ValidationResult {
	return agent.ValidationResult{StageID: "unit", Passed: false}
}

func (deterministicValidation) Current(
	context.Context,
	domain.Run,
) agent.ValidationResult {
	return agent.ValidationResult{StageID: "unit", Passed: true}
}

func (deterministicValidation) Final(
	context.Context,
	domain.Run,
) agent.ValidationResult {
	return agent.ValidationResult{StageID: "full", Passed: true}
}

type demoApplication struct{ target *store.MemoryStore }

func (*demoApplication) CreateRun(context.Context, app.CreateRunRequest) (domain.Run, error) {
	return domain.Run{}, errors.New("demo run creation is unavailable")
}
func (a *demoApplication) GetRun(
	ctx context.Context,
	runID domain.RunID,
) (domain.Run, error) {
	return a.target.GetRun(ctx, runID)
}
func (*demoApplication) CancelRun(context.Context, domain.RunID) error {
	return errors.New("demo cancellation is unavailable")
}
func (*demoApplication) Approve(context.Context, domain.RunID, string) error {
	return errors.New("demo approval is unavailable")
}
func (*demoApplication) Reject(context.Context, domain.RunID, string, bool) error {
	return errors.New("demo rejection is unavailable")
}
func (*demoApplication) Preflight(
	context.Context,
	app.CreateRunRequest,
) app.PreflightReport {
	return app.PreflightReport{}
}

func localRunRequest(root string) app.CreateRunRequest {
	return app.CreateRunRequest{
		RepoRoot: root, Task: "Repair the deterministic check",
		Provider: "openai", Model: "mock-v1", Endpoint: "https://api.openai.com",
		Profile: domain.ProfileSupervised,
	}
}

func newLocalApplication(
	dataDir string,
	redactor *feedback.Redactor,
) (*app.Service, *store.MemoryStore, *credentials.Service, error) {
	target := store.NewMemory()
	credentialService := credentials.NewService(credentials.NewMemoryStore(), nil)
	if err := credentialService.Add(context.Background(), credentials.Ref{
		Provider: "openai", Host: "api.openai.com",
	}, []byte("e2e-only-credential")); err != nil {
		return nil, nil, nil, err
	}
	factory := app.AgentLoopFactory(func(
		_ context.Context,
		_ app.RunSetup,
	) (*agent.Loop, *policy.ApprovalStore, error) {
		var decisions atomic.Int32
		mockProvider := provider.NewMock(func(
			context.Context,
			provider.Request,
		) (provider.Response, error) {
			var action domain.Action
			if decisions.Add(1) == 1 {
				action = domain.Action{
					Kind: "apply_patch",
					Args: json.RawMessage(
						`{"patch":"--- a/` + approvalCanary + `.txt\n+++ b/` +
							approvalCanary + `.txt\n@@ -1 +1 @@\n-old\n+new\n"}`,
					),
				}
			} else {
				action = domain.Action{Kind: "finish", Args: json.RawMessage(`{}`)}
			}
			return provider.Response{Decision: domain.AgentDecision{
				Version: "1", Action: action,
			}}, nil
		})
		approvals := policy.NewApprovalStore()
		loop := agent.New(agent.Dependencies{
			Store: target, Provider: mockProvider, Actions: deterministicActions{},
			Policy: policy.NewEngine(), Feedback: feedback.Pipeline{},
			Validation: deterministicValidation{}, Approvals: approvals,
			Budget: budget.New(budget.Limits{
				MaxDecisions: 4, MaxMutations: 2, MaxProtocolRepairs: 1,
				WallClock: time.Minute,
			}, realClock{}),
		})
		return loop, approvals, nil
	})
	var runSequence atomic.Uint64
	application, err := app.NewService(context.Background(), app.Options{
		Store: target, Loops: app.NewAgentLoopController(factory, *redactor),
		Credentials: credentialService, DataDir: dataDir,
		Locks: app.NewRepoLocksAt(filepath.Join(dataDir, "repo-locks")),
		NewRunID: func() domain.RunID {
			return domain.RunID(fmt.Sprintf("run-created-%d", runSequence.Add(1)))
		},
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return application, target, credentialService, nil
}

func prepareRepository(root string) (string, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	configText := `version = 1
default_profile = "supervised"

[[validation]]
id = "unit"
kind = "targeted-test"
executable = "git"
args = ["--version"]
working_directory = "."
timeout = "30s"
max_output_bytes = 4096
required = true
`
	for path, content := range map[string]string{
		".ai4se-harness.toml": configText,
		"a.txt":               "old\n",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o600); err != nil {
			return "", err
		}
	}
	for _, args := range [][]string{
		{"init"}, {"config", "user.name", "AI4SE E2E"},
		{"config", "user.email", "e2e@example.invalid"},
		{"add", "."}, {"commit", "-m", "e2e baseline"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git %v: %w: %s", args, err, output)
		}
	}
	return filepath.Clean(root), nil
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
	tempRoot, err := os.MkdirTemp("", "ai4se-e2e-")
	if err != nil {
		panic(err)
	}
	repoRoot, err := prepareRepository(filepath.Join(tempRoot, "repo"))
	if err != nil {
		panic(err)
	}
	centralRedactor := feedback.NewRedactor(nil)
	localApp, localStore, credentialService, err := newLocalApplication(
		filepath.Join(tempRoot, "data"), &centralRedactor,
	)
	if err != nil {
		panic(err)
	}

	random := &deterministicReader{}
	var localRouter *httpapi.Router
	localShell := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		serveIndex(writer, index, runtimeScript(localRouter.CSRFToken(), false, nil))
	})
	localRouter, err = httpapi.NewLocal(httpapi.Options{
		Application: localApp, Store: localStore, Credentials: credentialService,
		Capabilities: httpapi.LocalCapabilities(), Host: localAddress, Random: random,
		AppShell: localShell, PollInterval: 10 * time.Millisecond,
		HeartbeatInterval: 100 * time.Millisecond, Redactor: &centralRedactor,
	})
	if err != nil {
		panic(err)
	}

	demoStore := store.NewMemory()
	seedDemo(demoStore)
	demoRouter, err := httpapi.NewDemo(httpapi.Options{
		Application: &demoApplication{target: demoStore}, Store: demoStore,
		Capabilities: httpapi.DemoCapabilities("demo-feedback"),
		PollInterval: 10 * time.Millisecond, HeartbeatInterval: 100 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}

	failures := &oneShotFailures{}
	localHandler := assetWrapper(dist, localRouter, nil, repoRoot, localStore, failures)
	demoShell := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		serveIndex(writer, index, runtimeScript("", true, []string{"demo-feedback"}))
	})
	demoHandler := assetWrapper(dist, demoRouter, demoShell, "", nil, nil)
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStartup()
	listenerConfig := net.ListenConfig{}
	localListener, err := listenerConfig.Listen(startupContext, "tcp", localAddress)
	if err != nil {
		panic(err)
	}
	demoListener, err := listenerConfig.Listen(startupContext, "tcp", demoAddress)
	if err != nil {
		_ = localListener.Close()
		panic(err)
	}
	cancelStartup()
	go func() {
		if serveErr := http.Serve(localListener, localHandler); serveErr != nil &&
			!errors.Is(serveErr, http.ErrServerClosed) {
			panic(serveErr)
		}
	}()
	if serveErr := http.Serve(demoListener, demoHandler); serveErr != nil {
		panic(serveErr)
	}
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

func assetWrapper(
	dist string,
	api http.Handler,
	shell http.Handler,
	repoRoot string,
	eventStore *store.MemoryStore,
	failures *oneShotFailures,
) http.Handler {
	files := http.FileServer(http.Dir(dist))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/healthz" {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if repoRoot != "" && request.URL.Path == "/e2e/repository" {
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(map[string]string{"repo_root": repoRoot})
			return
		}
		if eventStore != nil && strings.HasPrefix(request.URL.Path, "/e2e/events/") {
			runID := domain.RunID(strings.TrimPrefix(request.URL.Path, "/e2e/events/"))
			events, listErr := eventStore.ListEvents(request.Context(), runID, 1)
			if listErr != nil {
				http.Error(writer, listErr.Error(), http.StatusInternalServerError)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(events)
			return
		}
		if failures != nil && request.URL.Path == "/e2e/fail-next" {
			var input struct {
				Target string `json:"target"`
			}
			if request.Method != http.MethodPost ||
				json.NewDecoder(request.Body).Decode(&input) != nil ||
				!failures.arm(input.Target) {
				http.Error(writer, "invalid one-shot failure target", http.StatusBadRequest)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		failureTarget := ""
		failureMessage := ""
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/runs":
			failureTarget, failureMessage = "runs", "Injected run-list failure"
		case request.Method == http.MethodGet &&
			strings.HasPrefix(request.URL.Path, "/api/v1/credentials/"):
			failureTarget, failureMessage = "credential", "Injected credential-status failure"
		}
		if failureTarget != "" && failures != nil && failures.consume(failureTarget) {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"error": map[string]string{
					"code": "E2E_INJECTED_FAILURE", "message": failureMessage,
					"request_id": "e2e-injected",
				},
			})
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
