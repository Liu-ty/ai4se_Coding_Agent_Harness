package budget

import "sync"

// ProgressOutcome describes whether an observation permits progress, records a
// warning, or requires the run to stop.
type ProgressOutcome string

const (
	ProgressContinue ProgressOutcome = "continue"
	ProgressWarning  ProgressOutcome = "warning"
	ProgressStop     ProgressOutcome = "stop"
)

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

// Observe records feedback and classifies the observed progress.
func (p *ProgressDetector) Observe(fingerprint, diffDigest string) ProgressOutcome {
	p.mu.Lock()
	defer p.mu.Unlock()

	if fingerprint == "" || diffDigest == "" {
		p.lastFingerprint = ""
		p.lastDiffDigest = ""
		p.consecutive = 0
		return ProgressContinue
	}

	if fingerprint == p.lastFingerprint && diffDigest == p.lastDiffDigest {
		p.consecutive++
		if p.consecutive >= p.threshold {
			return ProgressStop
		}
		return ProgressContinue
	}

	outcome := ProgressContinue
	if fingerprint == p.lastFingerprint && p.lastDiffDigest != "" {
		outcome = ProgressWarning
	}
	p.lastFingerprint = fingerprint
	p.lastDiffDigest = diffDigest
	p.consecutive = 1

	if p.consecutive >= p.threshold {
		return ProgressStop
	}
	return outcome
}
