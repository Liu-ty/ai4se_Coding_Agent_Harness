package storeport_test

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

var _ storeport.Store = (*recordingStore)(nil)

func TestStorePortPackageHasNoConcreteStoreImports(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) == "storeport_test.go" {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range parsed.Imports {
			switch imported.Path.Value {
			case `"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"`, `"database/sql"`, `"embed"`, `"modernc.org/sqlite"`:
				t.Fatalf("%s imports concrete store dependency %s", file, imported.Path.Value)
			}
		}
	}
}

type recordingStore struct{}

func (*recordingStore) CreateRun(context.Context, domain.Run) error { return nil }

func (*recordingStore) CreateRunWithEvent(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error) {
	return domain.RunEvent{}, nil
}

func (*recordingStore) UpdateRun(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error) {
	return domain.RunEvent{}, nil
}

func (*recordingStore) UpdateRunIfState(context.Context, domain.Run, domain.RunState, string, json.RawMessage) (domain.RunEvent, error) {
	return domain.RunEvent{}, nil
}

func (*recordingStore) AppendEvent(context.Context, domain.RunID, string, json.RawMessage) (domain.RunEvent, error) {
	return domain.RunEvent{}, nil
}

func (*recordingStore) GetRun(context.Context, domain.RunID) (domain.Run, error) {
	return domain.Run{}, nil
}

func (*recordingStore) ListRuns(context.Context) ([]domain.Run, error) { return nil, nil }

func (*recordingStore) ListEvents(context.Context, domain.RunID, uint64) ([]domain.RunEvent, error) {
	return nil, nil
}

func (*recordingStore) PutArtifact(context.Context, domain.Artifact) error { return nil }
