package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/budget"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/demo"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/testutil/testrepo"
)

func TestDemoCompositionExcludesRealCapabilities(t *testing.T) {
	composition, err := demo.NewComposition(context.Background(), "127.0.0.1:4319")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/api/v1/credentials/openai/api.openai.com",
		"/api/v1/config/validate",
		"/api/v1/runs/new-run",
	} {
		recorder := httptest.NewRecorder()
		composition.Router().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, recorder.Code)
		}
	}
}

func TestLocalRuntimeUsesAppCompositionForValidatedPreflight(t *testing.T) {
	repo := testrepo.New(t, map[string]string{
		"a.txt": "old\n",
		".ai4se-harness.toml": fmt.Sprintf(`version = 1
default_profile = "review"

[[validation]]
id = "unit"
kind = "test"
executable = %q
args = ["--version"]
working_directory = "."
timeout = "30s"
max_output_bytes = 4096
required = true
`, "git"),
	})
	credentialService := credentials.NewService(credentials.NewMemoryStore(), nil)
	if err := credentialService.Add(t.Context(), credentials.Ref{Provider: "openai", Host: "api.openai.com"}, []byte("test-key")); err != nil {
		t.Fatal(err)
	}
	runtime, err := newLocalRuntime(t.Context(), repo.Root, localRuntimeOptions{dataDir: t.TempDir(), credentials: credentialService})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	report := runtime.Application.Preflight(t.Context(), localRunRequest(repo.Root, "repair", ".ai4se-harness.toml"))
	if !report.OK {
		t.Fatalf("preflight = %#v", report)
	}
}

func TestLocalRuntimeCreatesItsDataDirectoryForACleanRepository(t *testing.T) {
	repo := testrepo.New(t, map[string]string{
		".ai4se-harness.toml": "version = 1\ndefault_profile = \\\"review\\\"\n",
		"a.txt":               "old\n",
	})
	dataDir := filepath.Join(repo.Root, ".ai4se-harness")
	runtime, err := newLocalRuntime(t.Context(), repo.Root, localRuntimeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if info, err := os.Stat(dataDir); err != nil || !info.IsDir() {
		t.Fatalf("data directory = %#v, %v", info, err)
	}
}

// Returning at the approval boundary, or serving it from a newly recovered
// service, must fail this test: only the creating controller owns the resumable
// loop session.
func TestContinueRunServesApprovalOnOwnedRuntimeUntilTerminal(t *testing.T) {
	fixture := newOwnedApprovalFixture(t)
	runtime, storage, run := fixture.runtime, fixture.storage, fixture.run

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	var output synchronizedBuffer
	done := make(chan error, 1)
	go func() {
		done <- continueRun(ctx, runtime, run.ID, &output, 3*time.Second)
	}()

	bootstrapURL := waitForContinuationURL(t, ctx, &output, done)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	response, err := client.Get(bootstrapURL)
	if err != nil {
		t.Fatal(err)
	}
	var runtimeConfig struct {
		CSRFToken string `json:"csrfToken"`
	}
	match := regexp.MustCompile(`window\.__AI4SE_RUNTIME__=([^;]+);`).FindSubmatch(responseBody(t, response))
	if len(match) != 2 || json.Unmarshal(match[1], &runtimeConfig) != nil || runtimeConfig.CSRFToken == "" {
		t.Fatalf("local runtime configuration missing from bootstrapped shell: %q", match)
	}

	var approval agent.ApprovalRequired
	deadline := time.Now().Add(time.Second)
	for approval.Digest == "" && time.Now().Before(deadline) {
		events, listErr := storage.ListEvents(ctx, run.ID, 1)
		if listErr != nil {
			t.Fatal(listErr)
		}
		for _, event := range events {
			if event.Type == "ApprovalRequired" {
				if err := json.Unmarshal(event.Payload, &approval); err != nil {
					t.Fatal(err)
				}
			}
		}
		if approval.Digest == "" {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if approval.Digest == "" {
		t.Fatal("approval request was not persisted")
	}
	baseURL := strings.Split(bootstrapURL, "/?bootstrap=")[0]
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(
		"%s/api/v1/runs/%s/approvals/%s/approve", baseURL, run.ID, approval.Digest,
	), strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", baseURL)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", runtimeConfig.CSRFToken)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if body := responseBody(t, response); response.StatusCode != http.StatusNoContent {
		t.Fatalf("approve status = %d: %s", response.StatusCode, body)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	stored, err := storage.GetRun(t.Context(), run.ID)
	if err != nil || stored.State != domain.StateSucceeded {
		t.Fatalf("continued run = %#v, %v", stored, err)
	}
	if _, err := client.Get(baseURL + "/healthz"); err == nil {
		t.Fatal("approval server still owns its listener after terminal state")
	}
}

func TestContinueRunStopsOwnedRunWhenApprovalWindowEnds(t *testing.T) {
	fixture := newOwnedApprovalFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	var output synchronizedBuffer
	err := continueRun(ctx, fixture.runtime, fixture.run.ID, &output, 50*time.Millisecond)
	if !errors.Is(err, ErrApprovalWindowEnded) {
		t.Fatalf("error = %v, want ErrApprovalWindowEnded", err)
	}
	stored, getErr := fixture.storage.GetRun(t.Context(), fixture.run.ID)
	if getErr != nil || stored.State != domain.StateStopped {
		t.Fatalf("timed-out run = %#v, %v", stored, getErr)
	}
	match := regexp.MustCompile(`continue at (http://[^/]+)`).FindStringSubmatch(output.String())
	if len(match) != 2 {
		t.Fatalf("continuation URL missing from %q", output.String())
	}
	if _, err := http.Get(match[1] + "/healthz"); err == nil {
		t.Fatal("timed-out approval server retained its listener")
	}
}

type ownedApprovalFixture struct {
	runtime *localRuntime
	storage *store.SQLiteStore
	run     domain.Run
}

func newOwnedApprovalFixture(t *testing.T) ownedApprovalFixture {
	t.Helper()
	repo := testrepo.New(t, map[string]string{
		"a.txt": "old\n",
		".ai4se-harness.toml": fmt.Sprintf(`version = 1
default_profile = "supervised"

[[validation]]
id = "unit"
kind = "test"
executable = %q
args = ["--version"]
working_directory = "."
timeout = "30s"
max_output_bytes = 4096
required = true
`, "git"),
	})
	dataDir := t.TempDir()
	storage, err := store.OpenSQLite(filepath.Join(dataDir, "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	credentialService := credentials.NewService(credentials.NewMemoryStore(), nil)
	if err := credentialService.Add(t.Context(), credentials.Ref{
		Provider: "openai", Host: "api.openai.com",
	}, []byte("test-key")); err != nil {
		t.Fatal(err)
	}
	patch := domain.Action{Kind: "apply_patch", Args: json.RawMessage(
		`{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`,
	)}
	decisions := 0
	factory := app.AgentLoopFactory(func(_ context.Context, _ app.RunSetup) (*agent.Loop, *policy.ApprovalStore, error) {
		decisionProvider := provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
			decisions++
			if decisions == 1 {
				return provider.Response{Decision: domain.AgentDecision{Version: "1", Action: patch}}, nil
			}
			return provider.Response{Decision: domain.AgentDecision{
				Version: "1", Action: domain.Action{Kind: "finish", Args: json.RawMessage(`{}`)},
			}}, nil
		})
		approvals := policy.NewApprovalStore()
		return agent.New(agent.Dependencies{
			Store: storage, Provider: decisionProvider, Actions: approvalAction{},
			Policy: policy.NewEngine(), Approvals: approvals, Feedback: feedback.Pipeline{},
			Validation: approvalChecks{}, Budget: budget.New(budget.Limits{
				MaxDecisions: 3, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute,
			}, timeClock{}),
		}), approvals, nil
	})
	redactor := feedback.NewRedactor(nil)
	application, err := app.NewLocal(t.Context(), storage, factory, credentialService, dataDir, &redactor)
	if err != nil {
		t.Fatal(err)
	}
	runtime := &localRuntime{Application: application, Store: storage, Credentials: credentialService}
	t.Cleanup(func() { _ = runtime.Close() })
	run, err := application.CreateRun(t.Context(), app.CreateRunRequest{
		RepoRoot: repo.Root, Task: "repair", Provider: "openai", Model: "test",
		Endpoint: "https://api.openai.com", Profile: domain.ProfileSupervised,
		ConfigPath: ".ai4se-harness.toml",
	})
	if err != nil {
		t.Fatal(err)
	}
	stateDeadline := time.Now().Add(2 * time.Second)
	for {
		stored, getErr := storage.GetRun(t.Context(), run.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if stored.State == domain.StateAwaitingApproval {
			break
		}
		if terminalRunState(stored.State) || time.Now().After(stateDeadline) {
			events, _ := storage.ListEvents(t.Context(), run.ID, 1)
			t.Fatalf("run did not reach approval: state=%s events=%#v", stored.State, events)
		}
		time.Sleep(10 * time.Millisecond)
	}

	return ownedApprovalFixture{runtime: runtime, storage: storage, run: run}
}

type approvalAction struct{}

func (approvalAction) Execute(context.Context, domain.Action) (agent.ActionResult, error) {
	return agent.ActionResult{DiffDigest: "changed"}, nil
}

type approvalChecks struct{}

func (approvalChecks) Baseline(context.Context, domain.Run) agent.ValidationResult {
	return agent.ValidationResult{StageID: "unit", Passed: false}
}
func (approvalChecks) Current(context.Context, domain.Run) agent.ValidationResult {
	return agent.ValidationResult{StageID: "unit", Passed: true}
}
func (approvalChecks) Final(context.Context, domain.Run) agent.ValidationResult {
	return agent.ValidationResult{StageID: "unit", Passed: true}
}

type synchronizedBuffer struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *synchronizedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(value)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}

func waitForContinuationURL(
	t *testing.T,
	ctx context.Context,
	output *synchronizedBuffer,
	done <-chan error,
) string {
	t.Helper()
	pattern := regexp.MustCompile(`continue at (http://\S+)`)
	for {
		if match := pattern.FindStringSubmatch(output.String()); len(match) == 2 {
			return match[1]
		}
		select {
		case err := <-done:
			t.Fatalf("run ownership ended before continuation URL: %v (output %q)", err, output.String())
		case <-ctx.Done():
			t.Fatalf("continuation URL not emitted: %q", output.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func responseBody(t *testing.T, response *http.Response) []byte {
	t.Helper()
	defer response.Body.Close()
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDemoCompositionRejectsEveryForbiddenRouteAndUsesNoLocalTypes(t *testing.T) {
	composition, err := demo.NewComposition(t.Context(), "127.0.0.1:4319")
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/credentials/openai/api.openai.com"},
		{http.MethodPut, "/api/v1/credentials/openai/api.openai.com"},
		{http.MethodPost, "/api/v1/runs"},
		{http.MethodPost, "/api/v1/config/validate"},
		{http.MethodGet, "/api/v1/runs/demo-feedback-loop/artifacts/diff"},
		{http.MethodPost, "/api/v1/runs/demo-feedback-loop/cancel"},
	} {
		recorder := httptest.NewRecorder()
		composition.Router().ServeHTTP(recorder, httptest.NewRequest(request.method, request.path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s %s = %d", request.method, request.path, recorder.Code)
		}
	}
	compositionType := reflect.TypeOf(composition).Elem()
	if compositionType.PkgPath() != "github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/demo" {
		t.Fatalf("unexpected demo composition type %T", composition)
	}
	for index := 0; index < compositionType.NumField(); index++ {
		fieldType := compositionType.Field(index).Type
		for fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if strings.Contains(fieldType.PkgPath(), "/credentials") || strings.Contains(fieldType.PkgPath(), "/executor") {
			t.Fatalf("demo stores forbidden capability field %q of type %s", compositionType.Field(index).Name, fieldType)
		}
	}
	if _, err := os.Stat(filepath.Join(".", ".ai4se-harness")); err == nil {
		t.Fatal("demo created a host runtime directory")
	}
}
