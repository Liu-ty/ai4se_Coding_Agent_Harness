// Package store persists run snapshots, immutable events, and artifacts.
package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var (
	ErrEmptyRunID      = errors.New("run ID is required")
	ErrEmptyEventType  = errors.New("event type is required")
	ErrEmptyArtifactID = errors.New("artifact ID is required")
	ErrRunNotFound     = errors.New("run not found")
)

// Store is the persistence port for a single harness run.
type Store interface {
	CreateRun(context.Context, domain.Run) error
	UpdateRun(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error)
	AppendEvent(context.Context, domain.RunID, string, json.RawMessage) (domain.RunEvent, error)
	GetRun(context.Context, domain.RunID) (domain.Run, error)
	ListEvents(context.Context, domain.RunID, uint64) ([]domain.RunEvent, error)
	PutArtifact(context.Context, domain.Artifact) error
}

func validateRunID(runID domain.RunID) error {
	if runID == "" {
		return ErrEmptyRunID
	}
	return nil
}

func validateEvent(runID domain.RunID, eventType string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if eventType == "" {
		return ErrEmptyEventType
	}
	return nil
}

func validateArtifact(artifact domain.Artifact) error {
	if artifact.ID == "" {
		return ErrEmptyArtifactID
	}
	return validateRunID(artifact.RunID)
}

func newEvent(runID domain.RunID, sequence uint64, eventType string, payload json.RawMessage, previousHash string, at time.Time) domain.RunEvent {
	payloadCopy := cloneJSON(payload)
	event := domain.RunEvent{
		RunID:        runID,
		Sequence:     sequence,
		Type:         eventType,
		At:           at.UTC(),
		Payload:      payloadCopy,
		PreviousHash: previousHash,
	}
	event.Hash = hashEvent(event)
	return event
}

func hashEvent(event domain.RunEvent) string {
	hash := sha256.New()
	writeField := func(value []byte) {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write(value)
	}
	var sequence [8]byte
	binary.BigEndian.PutUint64(sequence[:], event.Sequence)
	var at [8]byte
	binary.BigEndian.PutUint64(at[:], uint64(event.At.UnixNano()))
	writeField([]byte(event.RunID))
	writeField(sequence[:])
	writeField([]byte(event.Type))
	writeField(at[:])
	writeField(event.Payload)
	writeField([]byte(event.PreviousHash))
	return hex.EncodeToString(hash.Sum(nil))
}

func cloneJSON(payload json.RawMessage) json.RawMessage {
	if payload == nil {
		return json.RawMessage{}
	}
	return append(json.RawMessage(nil), payload...)
}

func cloneEvent(event domain.RunEvent) domain.RunEvent {
	event.Payload = cloneJSON(event.Payload)
	return event
}

func cloneArtifact(artifact domain.Artifact) domain.Artifact {
	if artifact.Content == nil {
		artifact.Content = []byte{}
		return artifact
	}
	artifact.Content = append([]byte(nil), artifact.Content...)
	return artifact
}
