package workspace_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/workspace"
)

func TestResolveRejectsParentEscape(t *testing.T) {
	_, err := workspace.Resolve(t.TempDir(), "../secret")
	if !errors.Is(err, workspace.ErrOutsideRoot) {
		t.Fatalf("got %v", err)
	}
}

func TestResolveRejectsProtectedPaths(t *testing.T) {
	for _, path := range []string{".git/config", ".env", "secrets/token.txt"} {
		_, err := workspace.Resolve(t.TempDir(), path)
		if !errors.Is(err, workspace.ErrProtectedPath) {
			t.Fatalf("Resolve(%q) error = %v", path, err)
		}
	}
}

func TestResolveRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs extra privileges on some Windows hosts")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	_, err := workspace.Resolve(root, "escape/file.txt")
	if !errors.Is(err, workspace.ErrOutsideRoot) {
		t.Fatalf("got %v", err)
	}
}
