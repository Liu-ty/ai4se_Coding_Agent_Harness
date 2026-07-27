package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/tools"
)

func TestCreateWritesNewFileAndReturnsDigest(t *testing.T) {
	root := t.TempDir()
	got, err := tools.NewCreateTool(root, 1024).Execute(context.Background(), json.RawMessage(`{"path":"new.txt","content":"new\n"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != "CREATE" || got.SHA256 == "" || string(got.Data) != `["new.txt"]` {
		t.Fatalf("result = %#v", got)
	}
	content, err := os.ReadFile(filepath.Join(root, "new.txt"))
	if err != nil || string(content) != "new\n" {
		t.Fatalf("content = %q, %v", content, err)
	}
}

func TestCreateNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := tools.NewCreateTool(root, 1024).Execute(context.Background(), json.RawMessage(`{"path":"a.txt","content":"new"}`))
	if !errors.Is(err, tools.ErrAlreadyExists) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateRejectsProtectedPathAndLimit(t *testing.T) {
	root := t.TempDir()
	tool := tools.NewCreateTool(root, 3)
	for _, test := range []struct {
		name, args string
		want       error
	}{
		{"protected", `{"path":".env","content":"x"}`, tools.ErrProtectedPath},
		{"limit", `{"path":"new.txt","content":"four"}`, tools.ErrCreateLimit},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), json.RawMessage(test.args))
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}
