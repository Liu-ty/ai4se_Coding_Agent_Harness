package budget

import "sync"

// ProgressDetector stops a run after repeated feedback with no observed progress.
// It is safe for concurrent callers to share.
type ProgressDetector struct {
	mu        sync.Mutex
	threshold int

	lastFingerprint string
	lastDiffDigest  string
	consecutive     int
}

// NewProgressDetector creates a detector with a minimum threshold of one.
func NewProgressDetector(threshold int) *ProgressDetector {
	if threshold <= 1 {
		threshold = 1
	}
	return &ProgressDetector{threshold: threshold}
}

// Observe records feedback. It returns true when the same complete pair has
// occurred at least threshold times consecutively.
func (p *ProgressDetector) Observe(fingerprint, diffDigest string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if fingerprint == "" || diffDigest == "" {
		p.lastFingerprint = ""
		p.lastDiffDigest = ""
		p.consecutive = 0
		return false
	}

	if fingerprint == p.lastFingerprint && diffDigest == p.lastDiffDigest {
		p.consecutive++
	} else {
		p.lastFingerprint = fingerprint
		p.lastDiffDigest = diffDigest
		p.consecutive = 1
	}

	return p.consecutive >= p.threshold
}
