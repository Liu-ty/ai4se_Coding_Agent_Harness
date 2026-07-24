package budget

import "testing"

func TestProgressDetectorStopsOnThresholdOfSameFailureAndDiff(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp", "diff-a") {
		t.Fatal("first observation must not stop")
	}
	if !p.Observe("fp", "diff-a") {
		t.Fatal("second identical observation must stop")
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

func TestProgressDetectorThresholdOneStopsOnFirstCompletePair(t *testing.T) {
	p := NewProgressDetector(1)
	if !p.Observe("fp", "diff") {
		t.Fatal("threshold one must stop on the first complete pair")
	}
}

func TestProgressDetectorEmptySignalsResetEvidence(t *testing.T) {
	p := NewProgressDetector(2)
	if p.Observe("fp", "diff") {
		t.Fatal("first observation must not stop")
	}
	if p.Observe("", "diff") || p.Observe("fp", "") {
		t.Fatal("incomplete signals must not stop")
	}
	if p.Observe("fp", "diff") {
		t.Fatal("incomplete signals must reset the prior evidence")
	}
	if !p.Observe("fp", "diff") {
		t.Fatal("second complete pair after reset must stop")
	}
}
