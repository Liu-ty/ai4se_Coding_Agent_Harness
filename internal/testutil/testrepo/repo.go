// Package testrepo creates committed repositories for integration tests.
package testrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type Repo struct{ Root string }

func New(t testing.TB, files map[string]string) *Repo {
	t.Helper()
	repo := &Repo{Root: t.TempDir()}
	for path, content := range files {
		repo.Write(t, path, content)
	}
	for _, args := range [][]string{{"init"}, {"config", "user.name", "AI4SE test"}, {"config", "user.email", "test@example.invalid"}, {"add", "."}, {"commit", "-m", "test baseline"}} {
		run(t, repo.Root, args...)
	}
	return repo
}

func (r *Repo) Read(t testing.TB, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(r.Root, path))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func (r *Repo) Write(t testing.TB, path, content string) {
	t.Helper()
	fullPath := filepath.Join(r.Root, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func run(t testing.TB, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
