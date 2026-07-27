package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

type ApprovalDigest string

// Digest hashes the exact canonical approval request with sorted baselines.
func Digest(runID domain.RunID, profile domain.PermissionProfile, action domain.Action, baselines map[string]string) ApprovalDigest {
	type baseline struct {
		Path string `json:"path"`
		Hash string `json:"hash"`
	}
	entries := make([]baseline, 0, len(baselines))
	for path, hash := range baselines {
		entries = append(entries, baseline{Path: path, Hash: hash})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	payload, _ := json.Marshal(struct {
		RunID     domain.RunID             `json:"run_id"`
		Profile   domain.PermissionProfile `json:"profile"`
		Action    domain.Action            `json:"action"`
		Baselines []baseline               `json:"baselines"`
	}{runID, profile, action, entries})
	sum := sha256.Sum256(payload)
	return ApprovalDigest(hex.EncodeToString(sum[:]))
}

type ApprovalStore struct {
	mu     sync.Mutex
	grants map[ApprovalDigest]struct{}
}

func NewApprovalStore() *ApprovalStore {
	return &ApprovalStore{grants: make(map[ApprovalDigest]struct{})}
}

func (s *ApprovalStore) Grant(digest ApprovalDigest) {
	if digest == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.grants == nil {
		s.grants = make(map[ApprovalDigest]struct{})
	}
	s.grants[digest] = struct{}{}
}

// Consume returns true once for a previously granted exact digest.
func (s *ApprovalStore) Consume(digest ApprovalDigest) bool {
	if digest == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.grants[digest]; !ok {
		return false
	}
	delete(s.grants, digest)
	return true
}
