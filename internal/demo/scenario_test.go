package demo

import (
	"context"
	"reflect"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func TestRunFeedbackLoopProducesTheFixedMechanismSequence(t *testing.T) {
	result, err := RunFeedbackLoop(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.State != domain.StateSucceeded {
		t.Fatalf("state = %q, want %q", result.State, domain.StateSucceeded)
	}
	wantEvents := []string{
		"PolicyDenied",
		"PatchApplied",
		"ValidationFailed",
		"FeedbackProduced",
		"PatchApplied",
		"ValidationPassed",
		"RunSucceeded",
	}
	if !reflect.DeepEqual(result.EventTypes(), wantEvents) {
		t.Fatalf("events = %v, want %v", result.EventTypes(), wantEvents)
	}
	if len(result.Actions) < 3 {
		t.Fatalf("actions = %d, want at least 3", len(result.Actions))
	}
	if result.Actions[1].Digest == result.Actions[2].Digest {
		t.Fatalf("patch digests must differ: %q", result.Actions[1].Digest)
	}
}
