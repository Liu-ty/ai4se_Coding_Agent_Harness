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
	for _, path := range []string{".git/config", "nested/.git/config", ".env", "secrets/token.txt"} {
		_, err := workspace.Resolve(t.TempDir(), path)
		if !errors.Is(err, workspace.ErrProtectedPath) {
			t.Fatalf("Resolve(%q) error = %v", path, err)
		}
	}
}

func TestResolveRejectsSymlinkToProtectedPath(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "vault"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "vault"), filepath.Join(root, "safe")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation is unavailable: %v", err)
		}
		t.Fatal(err)
	}
	_, err := workspace.Resolve(root, "safe/token.txt")
	if !errors.Is(err, workspace.ErrProtectedPath) {
		t.Fatalf("got %v", err)
	}
}

func TestResolveRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation is unavailable: %v", err)
		}
		t.Fatal(err)
	}
	_, err := workspace.Resolve(root, "escape/file.txt")
	if !errors.Is(err, workspace.ErrOutsideRoot) {
		t.Fatalf("got %v", err)
	}
}
