package domain

import (
	"encoding/json"
	"time"
)

// RunEvent is an immutable, hash-linked record of a run change.
type RunEvent struct {
	RunID        RunID
	Sequence     uint64
	Type         string
	At           time.Time
	Payload      json.RawMessage
	PreviousHash string
	Hash         string
}

// Artifact is content produced while executing a run.
type Artifact struct {
	ID        string
	RunID     RunID
	Kind      string
	SHA256    string
	Content   []byte
	Truncated bool
}
