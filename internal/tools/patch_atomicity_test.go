package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/testutil/testrepo"
)

type gitRunnerFunc func(context.Context, string, []string, []byte) ([]byte, error)

func (f gitRunnerFunc) run(ctx context.Context, root string, args []string, input []byte) ([]byte, error) {
	return f(ctx, root, args, input)
}

func TestPatchReportsAtomicityBreachAfterInjectedApplyMutation(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "old\n"})
	runner := gitRunnerFunc(func(ctx context.Context, root string, args []string, input []byte) ([]byte, error) {
		if len(args) == 3 && args[0] == "apply" && args[1] == "--whitespace=nowarn" {
			if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("broken\n"), 0600); err != nil {
				t.Fatal(err)
			}
			return nil, errors.New("injected apply failure")
		}
		return runGit(ctx, root, args, input)
	})
	tool := newPatchTool(repo.Root, PatchLimits{MaxFiles: 5, MaxChangedLines: 500}, runner)
	args := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`

	_, err := tool.Execute(context.Background(), json.RawMessage(args))
	if !errors.Is(err, ErrPatchAtomicityBreach) {
		t.Fatalf("got %v", err)
	}
}
