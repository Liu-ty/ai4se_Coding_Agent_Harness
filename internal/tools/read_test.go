package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/tools"
)

func TestReadReturnsHashAndTruncation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := tools.NewReadTool(root, 3).Execute(context.Background(), json.RawMessage(`{"path":"a.txt"}`))
	if err != nil || !got.Truncated || got.SHA256 == "" || got.Text != "abc" {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestReadTruncatesAtUTF8RuneBoundary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "unicode.txt"), []byte("a\u4e2db"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := tools.NewReadTool(root, 2).Execute(context.Background(), json.RawMessage(`{"path":"unicode.txt"}`))
	if err != nil || !got.Truncated || got.Text != "a" {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestReadRejectsBinaryAndProtectedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "binary.dat"), []byte{'a', 0, 'b'}, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := tools.NewReadTool(root, 100).Execute(context.Background(), json.RawMessage(`{"path":"binary.dat"}`))
	if !errors.Is(err, tools.ErrBinaryFile) {
		t.Fatalf("binary error = %v", err)
	}
	_, err = tools.NewReadTool(root, 100).Execute(context.Background(), json.RawMessage(`{"path":".env"}`))
	if !errors.Is(err, tools.ErrProtectedPath) {
		t.Fatalf("protected error = %v", err)
	}
}

func TestListAndSearchAreBoundedAndSorted(t *testing.T) {
	root := t.TempDir()
	for name, text := range map[string]string{"z.txt": "needle", "a.txt": "needle", "skip.bin": "x\x00y"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	list, err := tools.NewListTool(root, 1).Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !list.Truncated {
		t.Fatalf("list = %#v, %v", list, err)
	}
	var paths []string
	if err := json.Unmarshal(list.Data, &paths); err != nil || len(paths) != 1 || paths[0] != "a.txt" {
		t.Fatalf("paths = %q, %v", paths, err)
	}
	search, err := tools.NewSearchTool(root, 1).Execute(context.Background(), json.RawMessage(`{"query":"needle"}`))
	if err != nil || !search.Truncated {
		t.Fatalf("search = %#v, %v", search, err)
	}
	var matches []tools.SearchMatch
	if err := json.Unmarshal(search.Data, &matches); err != nil || len(matches) != 1 || matches[0].Path != "a.txt" {
		t.Fatalf("matches = %#v, %v", matches, err)
	}
}

func TestListReturnsGloballySortedPrefixForNestedPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "a"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"a/z.txt": "nested", "a.txt": "top-level"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := tools.NewListTool(root, 1).Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !got.Truncated {
		t.Fatalf("%#v %v", got, err)
	}
	var paths []string
	if err := json.Unmarshal(got.Data, &paths); err != nil || len(paths) != 1 || paths[0] != "a.txt" {
		t.Fatalf("paths = %q, %v", paths, err)
	}
}

func TestListStopsBeforeUnreadableLaterDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs, not chmod bits, control directory traversal")
	}
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	blocked := filepath.Join(root, "z-blocked")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0700) })
	if _, err := os.ReadDir(blocked); err == nil {
		t.Skip("test process can traverse chmod-restricted directories")
	}

	got, err := tools.NewListTool(root, 1).Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !got.Truncated {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestSearchRejectsInvalidRegex(t *testing.T) {
	_, err := tools.NewSearchTool(t.TempDir(), 10).Execute(context.Background(), json.RawMessage(`{"query":"["}`))
	if !errors.Is(err, tools.ErrInvalidRegex) {
		t.Fatalf("error = %v", err)
	}
}

func TestRegistryDispatchesKnownToolsOnly(t *testing.T) {
	registry, err := tools.NewRegistry(tools.NewListTool(t.TempDir(), 10))
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.Execute(context.Background(), domain.Action{Kind: "missing", Args: json.RawMessage(`{}`)})
	if !errors.Is(err, tools.ErrUnknownTool) {
		t.Fatalf("error = %v", err)
	}
}
