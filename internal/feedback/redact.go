package feedback

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b([A-Z0-9_]*(?:KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)[A-Z0-9_]*)=([^\s]+)`),
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{8,}`),
}

type Redactor struct {
	secrets []string
}

func NewRedactor(secrets []string) Redactor {
	copied := make([]string, 0, len(secrets)*2)
	seen := make(map[string]struct{}, len(secrets)*2)
	add := func(secret string) {
		if secret == "" {
			return
		}
		if _, exists := seen[secret]; exists {
			return
		}
		seen[secret] = struct{}{}
		copied = append(copied, secret)
	}
	for _, secret := range secrets {
		add(secret)
		encoded, err := json.Marshal(secret)
		if err == nil && len(encoded) >= 2 {
			add(string(encoded[1 : len(encoded)-1]))
		}
	}
	sort.Slice(copied, func(i, j int) bool { return len(copied[i]) > len(copied[j]) })
	return Redactor{secrets: copied}
}

func (r Redactor) Redact(text string) string {
	for _, secret := range r.secrets {
		text = strings.ReplaceAll(text, secret, "[REDACTED]")
	}
	text = secretPatterns[0].ReplaceAllString(text, `$1=[REDACTED]`)
	for _, pattern := range secretPatterns[1:] {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}
	return text
}
