package feedback

import (
	"testing"
	"unicode/utf8"
)

func TestSummarizeTruncatesAtUTF8Boundary(t *testing.T) {
	prefix := "VALIDATION_FAILED: "
	tests := []struct {
		name     string
		text     string
		maxBytes int
		want     string
	}{
		{name: "Chinese split", text: "你好", maxBytes: len(prefix) + 1, want: prefix},
		{name: "emoji split", text: "ok🙂tail", maxBytes: len(prefix) + len("ok") + 1, want: prefix + "ok"},
		{name: "exact character boundary", text: "你tail", maxBytes: len(prefix) + len("你"), want: prefix + "你"},
		{name: "zero limit", text: "你好", maxBytes: 0, want: ""},
		{name: "negative limit", text: "你好", maxBytes: -1, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, truncated := summarize("VALIDATION_FAILED", tt.text, tt.maxBytes)
			if !truncated {
				t.Fatalf("truncated = false, want true; summary=%q", got)
			}
			if got != tt.want {
				t.Fatalf("summary = %q, want %q", got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("summary is invalid UTF-8: %q", got)
			}
			if tt.maxBytes >= 0 && len([]byte(got)) > tt.maxBytes {
				t.Fatalf("summary byte length = %d, want <= %d", len([]byte(got)), tt.maxBytes)
			}
		})
	}
}
