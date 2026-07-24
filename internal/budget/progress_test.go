package budget

import (
	"sync"
	"testing"
)

func TestProgressDetectorStopsOnThresholdOfSameFailureAndDiff(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp", "diff-a") {
		t.Fatal("first observation must not stop")
	}
	if !p.Observe("fp", "diff-a") {
		t.Fatal("second identical observation must stop")
	}
}

func TestProgressDetectorStaysStoppedOnThirdIdenticalPair(t *testing.T) {
	p := NewProgressDetector(2)
	p.Observe("fp", "diff")
	p.Observe("fp", "diff")
	if !p.Observe("fp", "diff") {
		t.Fatal("a third identical pair must remain stopped")
	}
}

func TestProgressDetectorResetsAfterProgress(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp", "diff-a") {
		t.Fatal("first observation must not stop")
	}
	if p.Observe("fp", "diff-b") {
		t.Fatal("changed diff is progress and must reset the count")
	}
	if !p.Observe("fp", "diff-b") {
		t.Fatal("repeated post-progress pair must stop at threshold")
	}
}

func TestProgressDetectorDoesNotStopOnSameFailureWithDifferentDiff(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp", "diff-a") {
		t.Fatal("first observation must not stop")
	}
	if p.Observe("fp", "diff-b") {
		t.Fatal("same failure with a changed diff must not stop")
	}
}

func TestProgressDetectorResetsOnDifferentFailureWithSameDiff(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp-a", "diff") {
		t.Fatal("first observation must not stop")
	}
	if p.Observe("fp-b", "diff") {
		t.Fatal("different failure must reset the count")
	}
	if !p.Observe("fp-b", "diff") {
		t.Fatal("repeated new pair must stop at threshold")
	}
}

func TestProgressDetectorNormalizesNonPositiveThresholds(t *testing.T) {
	for _, threshold := range []int{0, -1, 1} {
		t.Run("threshold", func(t *testing.T) {
			p := NewProgressDetector(threshold)
			if !p.Observe("fp", "diff") {
				t.Fatalf("threshold %d must stop on the first complete pair", threshold)
			}
		})
	}
}

func TestProgressDetectorEmptyFingerprintResetsEvidence(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp", "diff") {
		t.Fatal("first observation must not stop")
	}
	if p.Observe("", "diff") {
		t.Fatal("empty fingerprint must not stop")
	}
	if p.Observe("fp", "diff") {
		t.Fatal("empty fingerprint must reset the prior evidence")
	}
	if !p.Observe("fp", "diff") {
		t.Fatal("second complete pair after an empty fingerprint must stop")
	}
}

func TestProgressDetectorEmptyDiffResetsEvidence(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp", "diff") {
		t.Fatal("first observation must not stop")
	}
	if p.Observe("fp", "") {
		t.Fatal("empty diff must not stop")
	}
	if p.Observe("fp", "diff") {
		t.Fatal("empty diff must reset the prior evidence")
	}
	if !p.Observe("fp", "diff") {
		t.Fatal("second complete pair after an empty diff must stop")
	}
}

func TestProgressDetectorChangedEvidenceAfterStopResets(t *testing.T) {
	p := NewProgressDetector(2)
	p.Observe("fp-a", "diff-a")
	p.Observe("fp-a", "diff-a")
	if p.Observe("fp-b", "diff-b") {
		t.Fatal("changed evidence after a stop must reset")
	}
	if !p.Observe("fp-b", "diff-b") {
		t.Fatal("second changed pair must stop")
	}
}

func TestProgressDetectorEmptyEvidenceAfterStopResets(t *testing.T) {
	p := NewProgressDetector(2)
	p.Observe("fp", "diff")
	p.Observe("fp", "diff")
	if p.Observe("", "diff") {
		t.Fatal("empty evidence after a stop must not stop")
	}
	if p.Observe("fp", "diff") {
		t.Fatal("empty evidence after a stop must reset")
	}
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
	if !p.Observe("fp", "diff") {
		t.Fatal("repeated complete evidence must remain stopped after concurrent observation")
	}
}
