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
)

// Store is the persistence port for a single harness run.
type Store interface {
	CreateRun(context.Context, domain.Run) error
	CreateRunWithEvent(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error)
	UpdateRun(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error)
	UpdateRunIfState(context.Context, domain.Run, domain.RunState, string, json.RawMessage) (domain.RunEvent, error)
	AppendEvent(context.Context, domain.RunID, string, json.RawMessage) (domain.RunEvent, error)
	GetRun(context.Context, domain.RunID) (domain.Run, error)
	ListRuns(context.Context) ([]domain.Run, error)
	ListEvents(context.Context, domain.RunID, uint64) ([]domain.RunEvent, error)
	PutArtifact(context.Context, domain.Artifact) error
	GetArtifact(context.Context, domain.RunID, string) (domain.Artifact, error)
}
