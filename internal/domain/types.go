package domain

import (
	"encoding/json"
	"errors"
	"time"
)

type RunID string
type RunState string

const (
	StateCreated            RunState = "CREATED"
	StatePreflight          RunState = "PREFLIGHT"
	StateBaselineValidating RunState = "BASELINE_VALIDATING"
	StateDeciding           RunState = "DECIDING"
	StateAwaitingApproval   RunState = "AWAITING_APPROVAL"
	StateExecuting          RunState = "EXECUTING"
	StateValidating         RunState = "VALIDATING"
	StateFinalValidating    RunState = "FINAL_VALIDATING"
	StateSucceeded          RunState = "SUCCEEDED"
	StateReviewComplete     RunState = "REVIEW_COMPLETE"
	StateStopped            RunState = "STOPPED"
)

type PermissionProfile string

const (
	ProfileReview        PermissionProfile = "review"
	ProfileSupervised    PermissionProfile = "supervised"
	ProfileWorkspaceAuto PermissionProfile = "workspace-auto"
)

type Action struct {
	Kind string          `json:"kind"`
	Args json.RawMessage `json:"args"`
}

type AgentDecision struct {
	Version         string `json:"version"`
	Action          Action `json:"action"`
	ExpectedOutcome string `json:"expected_outcome"`
}

type Observation struct {
	Code      string
	ExitCode  *int
	Stdout    string
	Stderr    string
	StartedAt time.Time
	EndedAt   time.Time
	Data      json.RawMessage
}

type Evidence struct {
	Source  string
	Message string
	Path    string
	Line    int
}

type StructuredFeedback struct {
	Category            string
	StageID             string
	Summary             string
	Fingerprint         string
	Evidence            []Evidence
	Retryable           bool
	OutputTruncated     bool
	PreviousOccurrences int
}

type Run struct {
	ID           RunID
	State        RunState
	Profile      PermissionProfile
	Task         string
	RepoRoot     string
	CurrentStage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var ErrInvalidTransition = errors.New("invalid run state transition")
