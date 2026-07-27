package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestPatchRestoresMutationWhenDiffFails(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "old\n"})
	runner := gitRunnerFunc(func(ctx context.Context, root string, args []string, input []byte) ([]byte, error) {
		switch {
		case reflect.DeepEqual(args, []string{"apply", "--check", "--whitespace=nowarn", "-"}):
			return nil, nil
		case reflect.DeepEqual(args, []string{"apply", "--whitespace=nowarn", "-"}):
			if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("new\n"), 0600); err != nil {
				t.Fatal(err)
			}
			return nil, nil
		case reflect.DeepEqual(args, []string{"diff", "--binary"}):
			return nil, errors.New("injected diff failure")
		case reflect.DeepEqual(args, []string{"apply", "-R", "--whitespace=nowarn", "-"}):
			if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("old\n"), 0600); err != nil {
				t.Fatal(err)
			}
			return nil, nil
		default:
			t.Fatalf("unexpected git args: %v", args)
			return nil, nil
		}
	})
	tool := newPatchTool(repo.Root, PatchLimits{MaxFiles: 5, MaxChangedLines: 500}, runner)
	args := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`

	_, err := tool.Execute(context.Background(), json.RawMessage(args))
	if !errors.Is(err, ErrPatchConflict) {
		t.Fatalf("got %v", err)
	}
	if got := repo.Read(t, "a.txt"); got != "old\n" {
		t.Fatalf("mutation remained after diff failure: %q", got)
	}
}

func TestPatchReportsAtomicityBreachWhenDiffFailureCannotBeRestored(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "old\n"})
	runner := gitRunnerFunc(func(ctx context.Context, root string, args []string, input []byte) ([]byte, error) {
		switch {
		case reflect.DeepEqual(args, []string{"apply", "--check", "--whitespace=nowarn", "-"}):
			return nil, nil
		case reflect.DeepEqual(args, []string{"apply", "--whitespace=nowarn", "-"}):
			if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("new\n"), 0600); err != nil {
				t.Fatal(err)
			}
			return nil, nil
		case reflect.DeepEqual(args, []string{"diff", "--binary"}):
			return nil, errors.New("injected diff failure")
		case reflect.DeepEqual(args, []string{"apply", "-R", "--whitespace=nowarn", "-"}):
			return nil, errors.New("injected reverse failure")
		default:
			t.Fatalf("unexpected git args: %v", args)
			return nil, nil
		}
	})
	tool := newPatchTool(repo.Root, PatchLimits{MaxFiles: 5, MaxChangedLines: 500}, runner)
	args := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`

	_, err := tool.Execute(context.Background(), json.RawMessage(args))
	if !errors.Is(err, ErrPatchAtomicityBreach) {
		t.Fatalf("got %v", err)
	}
}
