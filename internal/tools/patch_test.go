package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/testutil/testrepo"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/tools"
)

func TestPatchAppliesAndReturnsSortedPathsAndDiffDigest(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "old\n", "b.txt": "old\n"})
	patch := "--- a/b.txt\n+++ b/b.txt\n@@ -1 +1 @@\n-old\n+new\n--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"

	got, err := tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles: 5, MaxChangedLines: 500}).Execute(context.Background(), json.RawMessage(`{"patch":`+jsonString(patch)+`}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != "PATCH" || got.SHA256 == "" {
		t.Fatalf("result = %#v", got)
	}
	var paths []string
	if err := json.Unmarshal(got.Data, &paths); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "a.txt" || paths[1] != "b.txt" {
		t.Fatalf("paths = %q", paths)
	}
	if strings.ReplaceAll(repo.Read(t, "a.txt"), "\r\n", "\n") != "new\n" || strings.ReplaceAll(repo.Read(t, "b.txt"), "\r\n", "\n") != "new\n" {
		t.Fatalf("patch was not applied: a=%q b=%q result=%#v", repo.Read(t, "a.txt"), repo.Read(t, "b.txt"), got)
	}
}

func TestPatchRejectsStaleBaselineWithoutMutation(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "old\n"})
	tool := tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles: 5, MaxChangedLines: 500})
	args := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n","baselines":{"a.txt":"deadbeef"}}`

	_, err := tool.Execute(context.Background(), json.RawMessage(args))
	if !errors.Is(err, tools.ErrStaleBaseline) {
		t.Fatalf("got %v", err)
	}
	if got := repo.Read(t, "a.txt"); got != "old\n" {
		t.Fatalf("mutated: %q", got)
	}
}

func TestPatchRejectsConflictsWithoutMutation(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "changed\n"})
	args := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`

	_, err := tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles: 5, MaxChangedLines: 500}).Execute(context.Background(), json.RawMessage(args))
	if !errors.Is(err, tools.ErrPatchConflict) {
		t.Fatalf("got %v", err)
	}
	if got := repo.Read(t, "a.txt"); got != "changed\n" {
		t.Fatalf("mutated: %q", got)
	}
}

func TestPatchAcceptsHeaderLikeLinesInsideHunkPayload(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "-- a/other\n"})
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n--- a/other\n+++ b/other\n"

	_, err := tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles: 5, MaxChangedLines: 500}).Execute(context.Background(), json.RawMessage(`{"patch":`+jsonString(patch)+`}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.ReplaceAll(repo.Read(t, "a.txt"), "\r\n", "\n"); got != "++ b/other\n" {
		t.Fatalf("content = %q", got)
	}
}

func TestPatchRejectsUnsafeFormsAndLimits(t *testing.T) {
	repo := testrepo.New(t, map[string]string{"a.txt": "old\n"})
	tool := tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles: 1, MaxChangedLines: 500})
	for _, test := range []struct {
		name, patch string
		want        error
	}{
		{"delete", "--- a/a.txt\n+++ /dev/null\n@@ -1 +0,0 @@\n-old\n", tools.ErrInvalidPatch},
		{"rename", "rename from a.txt\nrename to b.txt\n--- a/a.txt\n+++ b/b.txt\n@@ -1 +1 @@\n-old\n+new\n", tools.ErrInvalidPatch},
		{"binary", "--- a/a.txt\n+++ b/a.txt\nGIT binary patch\n", tools.ErrInvalidPatch},
		{"protected", "--- a/.env\n+++ b/.env\n@@ -1 +1 @@\n-old\n+new\n", tools.ErrProtectedPath},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), json.RawMessage(`{"patch":`+jsonString(test.patch)+`}`))
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	_, err := tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles: 5, MaxChangedLines: 1}).Execute(context.Background(), json.RawMessage(`{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`))
	if !errors.Is(err, tools.ErrPatchLimit) {
		t.Fatalf("line limit error = %v", err)
	}
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n--- a/b.txt\n+++ b/b.txt\n@@ -1 +1 @@\n-old\n+new\n"
	_, err = tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles: 1, MaxChangedLines: 500}).Execute(context.Background(), json.RawMessage(`{"patch":`+jsonString(patch)+`}`))
	if !errors.Is(err, tools.ErrPatchLimit) {
		t.Fatalf("file limit error = %v", err)
	}
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
