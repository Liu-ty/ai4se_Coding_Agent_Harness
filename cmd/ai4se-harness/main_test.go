package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/demo"
)

func TestDemoCompositionExcludesRealCapabilities(t *testing.T) {
	composition, err := demo.NewComposition(context.Background(), "127.0.0.1:4319")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := composition.ProviderType(), "mock"; got != want {
		t.Fatalf("provider = %q, want %q", got, want)
	}
	if got, want := composition.ExecutorType(), "in-memory"; got != want {
		t.Fatalf("executor = %q, want %q", got, want)
	}
	if got, want := composition.RegisteredTools(), []string{"apply_patch", "finish"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tools = %v, want %v", got, want)
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
