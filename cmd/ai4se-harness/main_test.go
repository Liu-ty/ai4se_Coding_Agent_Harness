package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/demo"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
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

func TestWaitForTerminalKeepsRuntimeAliveUntilTheRunStops(t *testing.T) {
	application := &terminalAfterReads{states: []domain.RunState{domain.StateDeciding, domain.StateExecuting, domain.StateSucceeded}}
	if err := waitForTerminal(t.Context(), application, "run"); err != nil {
		t.Fatal(err)
	}
	if application.reads != 3 {
		t.Fatalf("reads = %d, want 3", application.reads)
	}
}

func TestWaitForTerminalReturnsApprovalHandoff(t *testing.T) {
	application := &terminalAfterReads{states: []domain.RunState{domain.StateAwaitingApproval}}
	if err := waitForTerminal(t.Context(), application, "run"); !errors.Is(err, ErrRunAwaitingApproval) {
		t.Fatalf("error = %v", err)
	}
}

type terminalAfterReads struct {
	states []domain.RunState
	reads  int
}

func (a *terminalAfterReads) GetRun(context.Context, domain.RunID) (domain.Run, error) {
	state := a.states[a.reads]
	a.reads++
	return domain.Run{State: state}, nil
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
