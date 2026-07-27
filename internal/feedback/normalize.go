package feedback

import (
	"regexp"
	"strings"
)

var (
	ansiPattern     = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	linePattern     = regexp.MustCompile(`:([0-9]+)(:| |\b)`)
	durationPattern = regexp.MustCompile(`\b[0-9]+(?:\.[0-9]+)?(?:ms|s)\b`)
	hexPattern      = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	uuidPattern     = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	tmpRootPattern  = regexp.MustCompile(`(?i)([A-Z]:)?[\\/](?:tmp|temp)[\\/][^\s]+`)
	spacePattern    = regexp.MustCompile(`[ \t]+`)
)

func normalizeHumanText(text string) string {
	text = ansiPattern.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func normalizeFingerprintText(text string) string {
	text = normalizeHumanText(text)
	text = strings.ReplaceAll(text, "\\", "/")
	text = linePattern.ReplaceAllString(text, ":<line>$2")
	text = durationPattern.ReplaceAllString(text, "<duration>")
	text = hexPattern.ReplaceAllString(text, "<hex>")
	text = uuidPattern.ReplaceAllString(text, "<uuid>")
	text = tmpRootPattern.ReplaceAllString(text, "<tmp>")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(spacePattern.ReplaceAllString(line, " "))
	}
	return strings.Join(lines, "\n")
}

func observationText(in Input) string {
	parts := []string{in.Observation.Stdout, in.Observation.Stderr, string(in.Observation.Data)}
	return strings.Trim(strings.Join(parts, "\n"), "\n")
}
