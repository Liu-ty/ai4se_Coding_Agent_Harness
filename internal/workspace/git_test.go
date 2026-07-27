package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/testutil/testrepo"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/workspace"
)

func TestGitBaselineHonorsCancelledContext(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "old\n"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := workspace.GitBaseline(ctx, repo.Root)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
