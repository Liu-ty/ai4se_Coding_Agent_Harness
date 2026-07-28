package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
)

const canaryKey = "CANARY_PROVIDER_KEY_DO_NOT_LOG"

type credentials struct{}

func (credentials) Get(context.Context, string, string) ([]byte, error) {
	return []byte(canaryKey), nil
}

func canonicalRequest() provider.Request {
	return provider.Request{Task: "repair", AllowedActions: []string{"read_file"}}
}
func canonicalDecision() string {
	return `{"version":"1","action":{"kind":"read_file","args":{"path":"bug.go"}},"expected_outcome":"inspect"}`
}
func assertDecision(t *testing.T, got provider.Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision.Version != "1" || got.Decision.Action.Kind != "read_file" {
		t.Fatalf("response=%#v", got)
	}
}

func TestOpenAIContractUsesCanonicalPathAndBearerAuthentication(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer "+canaryKey {
			t.Fatalf("request path/auth = %s/%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": canonicalDecision()}}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 5}})
	}))
	defer srv.Close()
	got, err := provider.NewOpenAI(srv.URL, srv.Client(), credentials{}).Decide(context.Background(), canonicalRequest())
	assertDecision(t, got, err)
}

func TestOpenAIDoesNotDuplicateV1EndpointPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": canonicalDecision()}}}})
	}))
	defer srv.Close()
	got, err := provider.NewOpenAI(srv.URL+"/v1", srv.Client(), credentials{}).Decide(context.Background(), canonicalRequest())
	assertDecision(t, got, err)
}

func TestAnthropicContractUsesCanonicalPathAndKeyHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != canaryKey || r.Header.Get("anthropic-version") == "" {
			t.Fatalf("request headers/path invalid")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]any{"type": "text", "text": canonicalDecision()}}, "usage": map[string]int{"input_tokens": 3, "output_tokens": 5}})
	}))
	defer srv.Close()
	got, err := provider.NewAnthropic(srv.URL, srv.Client(), credentials{}).Decide(context.Background(), canonicalRequest())
	assertDecision(t, got, err)
}

func TestProviderErrorsDoNotLeakCredential(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server rejected "+canaryKey, http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := provider.NewOpenAI(srv.URL, srv.Client(), credentials{}).Decide(context.Background(), canonicalRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if contains := stringify(err); contains != "" {
		t.Fatalf("error leaked credential: %q", contains)
	}
}

func TestProviderRejectsMalformedAndOversizedResponses(t *testing.T) {
	for _, body := range []string{"not-json", strings.Repeat("x", 1<<20+1)} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(body))
		}))
		_, err := provider.NewOpenAI(srv.URL, srv.Client(), credentials{}).Decide(context.Background(), canonicalRequest())
		srv.Close()
		if !errors.Is(err, provider.ErrInvalidResponse) {
			t.Fatalf("error=%v", err)
		}
	}
}
func stringify(err error) string {
	if text := err.Error(); len(text) > 0 && (containsText(text, canaryKey) || containsText(text, "server rejected")) {
		return text
	}
	return ""
}
func containsText(s, needle string) bool {
	return len(needle) > 0 && len(s) >= len(needle) && json.Valid(domain.Action{Args: json.RawMessage(`{}`)}.Args) && contains(s, needle)
}
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
