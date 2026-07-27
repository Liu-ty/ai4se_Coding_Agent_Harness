package feedback_test

import (
	"strings"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/executor"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
)

func TestFingerprintIgnoresANSIPathsTimingAndAddresses(t *testing.T) {
	a := "\x1b[31mFAIL pkg/a_test.go:12 took 1.24s ptr=0xabc123\x1b[0m"
	b := "FAIL pkg/a_test.go:99 took 9.87s ptr=0xdef456"
	if feedback.Fingerprint("unit-test", "TEST_FAILURE", a) != feedback.Fingerprint("unit-test", "TEST_FAILURE", b) {
		t.Fatal("unstable fingerprint")
	}
}

func TestRedactorRemovesKnownAndPatternSecrets(t *testing.T) {
	r := feedback.NewRedactor([]string{"exact-canary"})
	got := r.Redact("Authorization: Bearer exact-canary OPENAI_API_KEY=CANARY_SECRET_DO_NOT_LOG_123456789 token=sk-live-1234567890abcdef")
	for _, leaked := range []string{"exact-canary", "CANARY_SECRET_DO_NOT_LOG", "sk-live"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("leaked secret marker %q in %q", leaked, got)
		}
	}
}

func TestProcessRedactsSecretsBeforeSummaryEvidenceAndFingerprint(t *testing.T) {
	got := feedback.Pipeline{}.Process(feedback.Input{
		StageID: "unit",
		Code:    executor.CodeExit,
		Observation: domain.Observation{
			Code:     executor.CodeExit,
			ExitCode: intp(1),
			Stdout:   "FAIL pkg/a_test.go:12 expected exact-canary",
		},
		Secrets: []string{"exact-canary"},
	})
	if strings.Contains(got.Summary, "exact-canary") || strings.Contains(got.Fingerprint, "exact-canary") {
		t.Fatalf("secret leaked in feedback: %#v", got)
	}
	if len(got.Evidence) == 0 || strings.Contains(got.Evidence[0].Message, "exact-canary") {
		t.Fatalf("secret leaked in evidence: %#v", got.Evidence)
	}
}

func TestProcessClassifiesRequiredCategories(t *testing.T) {
	tests := []struct {
		name string
		in   feedback.Input
		want string
	}{
		{name: "invalid JSON", in: input("parse", "INVALID_JSON", "", 0), want: "PROTOCOL_ERROR"},
		{name: "unknown action", in: input("policy", "UNKNOWN_ACTION", "", 0), want: "POLICY_DENIED"},
		{name: "approval required", in: input("policy", "APPROVAL_REQUIRED", "", 0), want: "APPROVAL_REQUIRED"},
		{name: "path denial", in: input("policy", "REPOSITORY_ESCAPE", "", 0), want: "POLICY_DENIED"},
		{name: "stale patch", in: input("patch", "STALE_PATCH", "", 0), want: "STALE_PATCH"},
		{name: "missing executable", in: input("go-test", executor.CodeStartError, "executable file not found in PATH", 0), want: "MISSING_EXECUTABLE"},
		{name: "timeout", in: input("go-test", executor.CodeTimeout, "", 0), want: "TIMEOUT"},
		{name: "cancellation", in: input("go-test", executor.CodeCancelled, "", 0), want: "CANCELLED"},
		{name: "test failure", in: input("unit", executor.CodeExit, "--- FAIL: TestThing\nexpected true", 1), want: "TEST_FAILURE"},
		{name: "compile failure", in: input("compile", executor.CodeExit, "syntax error: unexpected name", 1), want: "COMPILE_FAILURE"},
		{name: "type failure", in: input("typecheck", executor.CodeExit, "cannot use string as int", 1), want: "TYPE_FAILURE"},
		{name: "lint failure", in: input("lint", executor.CodeExit, "golangci-lint found issues", 1), want: "LINT_FAILURE"},
		{name: "build failure", in: input("build", executor.CodeExit, "BUILD FAILED", 1), want: "BUILD_FAILURE"},
		{name: "generic exit", in: input("validation", executor.CodeExit, "command exited with status 2", 2), want: "VALIDATION_FAILED"},
		{name: "empty patch", in: input("patch", "EMPTY_PATCH", "", 0), want: "EMPTY_PATCH"},
		{name: "regression", in: input("final", "REGRESSION", "previously passing stage failed", 0), want: "REGRESSION"},
		{
			name: "configured validation regex",
			in: feedback.Input{
				StageID: "custom",
				Code:    executor.CodeExit,
				Observation: domain.Observation{
					Code:     executor.CodeExit,
					ExitCode: intp(1),
					Stderr:   "custom sentinel",
				},
				Rules: []config.ClassifierRule{{Category: "CUSTOM_FAILURE", Pattern: "custom sentinel"}},
			},
			want: "CUSTOM_FAILURE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := feedback.Pipeline{}.Process(tt.in)
			if got.Category != tt.want {
				t.Fatalf("category = %q, want %q; feedback=%#v", got.Category, tt.want, got)
			}
			if got.StageID != tt.in.StageID || got.Fingerprint == "" || got.PreviousOccurrences != tt.in.PriorOccurrences {
				t.Fatalf("incomplete feedback: %#v", got)
			}
		})
	}
}

func TestProcessBoundsEvidenceWithFirstAndLastLines(t *testing.T) {
	text := strings.Join([]string{"line-1", "line-2", "line-3", "line-4", "line-5"}, "\n")
	got := feedback.Pipeline{MaxEvidence: 4, MaxSummaryBytes: 16}.Process(input("unit", executor.CodeExit, text, 0))
	if !got.OutputTruncated {
		t.Fatalf("truncation not marked: %#v", got)
	}
	if len(got.Evidence) != 4 {
		t.Fatalf("evidence count = %d, want 4: %#v", len(got.Evidence), got.Evidence)
	}
	if got.Evidence[0].Message != "line-1" || got.Evidence[3].Message != "line-5" {
		t.Fatalf("first/last evidence not preserved: %#v", got.Evidence)
	}
	if len(got.Summary) > 16 {
		t.Fatalf("summary not bounded: %q", got.Summary)
	}
}

func TestProcessPreservesExecutorOutputTruncation(t *testing.T) {
	got := feedback.Pipeline{MaxEvidence: 100, MaxSummaryBytes: 100}.Process(feedback.Input{
		StageID: "unit",
		Code:    executor.CodeExit,
		Observation: domain.Observation{
			Code:            executor.CodeExit,
			ExitCode:        intp(1),
			Stdout:          "short output",
			OutputTruncated: true,
		},
	})
	if !got.OutputTruncated {
		t.Fatalf("executor truncation not preserved: %#v", got)
	}
}

func input(stageID, code, text string, exitCode int) feedback.Input {
	return feedback.Input{
		StageID: stageID,
		Code:    code,
		Observation: domain.Observation{
			Code:     code,
			ExitCode: intp(exitCode),
			Stdout:   text,
		},
		PriorOccurrences: exitCode,
	}
}

func intp(v int) *int { return &v }
