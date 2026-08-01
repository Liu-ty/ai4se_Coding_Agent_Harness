package agent

import (
	"testing"
	"unicode/utf8"
)

func TestBoundedPreservesUTF8AtByteLimit(t *testing.T) {
	got := bounded("abcd界", 5)
	if got != "abcd" || !utf8.ValidString(got) {
		t.Fatalf("bounded unicode = %q, want valid %q", got, "abcd")
	}
}
