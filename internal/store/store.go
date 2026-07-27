// Package store persists run snapshots, immutable events, and artifacts.
package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

// Clock supplies event timestamps at store construction boundaries.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func validateRunID(runID domain.RunID) error {
	if runID == "" {
		return storeport.ErrEmptyRunID
	}
	return nil
}

func validateEvent(runID domain.RunID, eventType string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if eventType == "" {
		return storeport.ErrEmptyEventType
	}
	return nil
}

func validateArtifact(artifact domain.Artifact) error {
	if artifact.ID == "" {
		return storeport.ErrEmptyArtifactID
	}
	return validateRunID(artifact.RunID)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
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

func normalizeRun(run domain.Run) domain.Run {
	run.CreatedAt = normalizeTime(run.CreatedAt)
	run.UpdatedAt = normalizeTime(run.UpdatedAt)
	return run
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC()
}
