package feedback

import (
	"strings"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func compressEvidence(text string, maxEvidence int) ([]domain.Evidence, bool) {
	lines := meaningfulLines(text)
	truncated := false
	if len(lines) > maxEvidence {
		truncated = true
		head := maxEvidence / 2
		tail := maxEvidence - head
		kept := make([]string, 0, maxEvidence)
		kept = append(kept, lines[:head]...)
		kept = append(kept, lines[len(lines)-tail:]...)
		lines = kept
	}
	out := make([]domain.Evidence, 0, len(lines))
	for _, line := range lines {
		out = append(out, domain.Evidence{Source: "observation", Message: line})
	}
	return out, truncated
}

func meaningfulLines(text string) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func summarize(category, text string, maxBytes int) (string, bool) {
	lines := meaningfulLines(text)
	summary := category
	if len(lines) > 0 {
		summary = category + ": " + lines[0]
	}
	if len(summary) <= maxBytes {
		return summary, false
	}
	if maxBytes <= 0 {
		return "", true
	}
	return summary[:maxBytes], true
}
