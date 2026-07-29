// Package storeport defines the persistence boundary consumed by harness core packages.
package storeport

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var (
	ErrEmptyRunID       = errors.New("run ID is required")
	ErrEmptyEventType   = errors.New("event type is required")
	ErrEmptyArtifactID  = errors.New("artifact ID is required")
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrRunAlreadyExists = errors.New("run already exists")
	ErrRunNotFound      = errors.New("run not found")
	ErrRunStateChanged  = errors.New("run state changed")
	ErrInvalidRunList   = errors.New("invalid run list query")
)

// RunListQuery selects a stable recent-runs page. A nil IDs slice means all
// runs; a non-nil empty slice means no runs. Limit zero is an internal
// unbounded query used by startup recovery.
type RunListQuery struct {
	Limit  int
	Offset int
	IDs    []domain.RunID
}

type RunPage struct {
	Runs    []domain.Run
	HasMore bool
}

// Store is the persistence port for a single harness run.
type Store interface {
	CreateRun(context.Context, domain.Run) error
	CreateRunWithEvent(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error)
	UpdateRun(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error)
	UpdateRunIfState(context.Context, domain.Run, domain.RunState, string, json.RawMessage) (domain.RunEvent, error)
	AppendEvent(context.Context, domain.RunID, string, json.RawMessage) (domain.RunEvent, error)
	GetRun(context.Context, domain.RunID) (domain.Run, error)
	ListRuns(context.Context, RunListQuery) (RunPage, error)
	ListEvents(context.Context, domain.RunID, uint64) ([]domain.RunEvent, error)
	PutArtifact(context.Context, domain.Artifact) error
	GetArtifact(context.Context, domain.RunID, string) (domain.Artifact, error)
}
