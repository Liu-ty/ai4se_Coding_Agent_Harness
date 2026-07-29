package feedback

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b([A-Z0-9_]*(?:KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)[A-Z0-9_]*)=([^\s]+)`),
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{8,}`),
}

type Redactor struct {
	state *redactorState
}

type redactorState struct {
	mu      sync.RWMutex
	secrets []string
}

func NewRedactor(secrets []string) Redactor {
	redactor := Redactor{state: &redactorState{}}
	for _, secret := range secrets {
		redactor.Register(secret)
	}
	return redactor
}

// Register adds a runtime-known secret. Copies of a Redactor created by
// NewRedactor share this registry, so composition can inject one redaction
// boundary into HTTP, app, and agent components.
func (r *Redactor) Register(secret string) {
	if secret == "" {
		return
	}
	if r.state == nil {
		r.state = &redactorState{}
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	copied := append([]string(nil), r.state.secrets...)
	seen := make(map[string]struct{}, len(copied)+2)
	for _, existing := range copied {
		seen[existing] = struct{}{}
	}
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
	add(secret)
	encoded, err := json.Marshal(secret)
	if err == nil && len(encoded) >= 2 {
		add(string(encoded[1 : len(encoded)-1]))
	}
	sort.Slice(copied, func(i, j int) bool { return len(copied[i]) > len(copied[j]) })
	r.state.secrets = copied
}

func (r Redactor) Redact(text string) string {
	var secrets []string
	if r.state != nil {
		r.state.mu.RLock()
		secrets = append(secrets, r.state.secrets...)
		r.state.mu.RUnlock()
	}
	for _, secret := range secrets {
		text = strings.ReplaceAll(text, secret, "[REDACTED]")
	}
	text = secretPatterns[0].ReplaceAllString(text, `$1=[REDACTED]`)
	for _, pattern := range secretPatterns[1:] {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}
	return text
}
