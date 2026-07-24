package budget

import (
	"sync"
	"testing"
)

func requireProgressOutcome(t *testing.T, got, want ProgressOutcome) {
	t.Helper()
	if got != want {
		t.Fatalf("outcome = %q, want %q", got, want)
	}
}

func TestProgressDetectorStopsOnThresholdOfSameFailureAndDiff(t *testing.T) {
	p := NewProgressDetector(2)
	requireProgressOutcome(t, p.Observe("fp", "diff-a"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", "diff-a"), ProgressStop)
}

func TestProgressDetectorStaysStoppedOnThirdIdenticalPair(t *testing.T) {
	p := NewProgressDetector(2)
	p.Observe("fp", "diff")
	p.Observe("fp", "diff")
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressStop)
}

func TestProgressDetectorWarnsOnSameFailureWithChangedDiff(t *testing.T) {
	p := NewProgressDetector(2)
	requireProgressOutcome(t, p.Observe("fp", "diff-a"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", "diff-b"), ProgressWarning)
	requireProgressOutcome(t, p.Observe("fp", "diff-b"), ProgressStop)
}

func TestProgressDetectorContinuesOnDifferentFailureWithSameDiff(t *testing.T) {
	p := NewProgressDetector(2)
	requireProgressOutcome(t, p.Observe("fp-a", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp-b", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp-b", "diff"), ProgressStop)
}

func TestProgressDetectorNormalizesNonPositiveThresholds(t *testing.T) {
	for _, threshold := range []int{0, -1, 1} {
		t.Run("threshold", func(t *testing.T) {
			p := NewProgressDetector(threshold)
			requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressStop)
		})
	}
}

func TestProgressDetectorEmptyFingerprintResetsEvidence(t *testing.T) {
	p := NewProgressDetector(2)
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressStop)
}

func TestProgressDetectorEmptyDiffResetsEvidence(t *testing.T) {
	p := NewProgressDetector(2)
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", ""), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressStop)
}

func TestProgressDetectorChangedEvidenceAfterStopResets(t *testing.T) {
	p := NewProgressDetector(2)
	p.Observe("fp-a", "diff-a")
	p.Observe("fp-a", "diff-a")
	requireProgressOutcome(t, p.Observe("fp-b", "diff-b"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp-b", "diff-b"), ProgressStop)
}

func TestProgressDetectorEmptyEvidenceAfterStopResets(t *testing.T) {
	p := NewProgressDetector(2)
	p.Observe("fp", "diff")
	p.Observe("fp", "diff")
	requireProgressOutcome(t, p.Observe("", "diff"), ProgressContinue)
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressContinue)
}

func TestProgressDetectorSupportsConcurrentObservation(t *testing.T) {
	p := NewProgressDetector(2)
	const observers = 32
	var workers sync.WaitGroup
	for range observers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_ = p.Observe("fp", "diff")
		}()
	}
	workers.Wait()
	requireProgressOutcome(t, p.Observe("fp", "diff"), ProgressStop)
}
