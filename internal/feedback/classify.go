package feedback

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/executor"
)

type Input struct {
	StageID          string
	Code             string
	Observation      domain.Observation
	Rules            []config.ClassifierRule
	PriorOccurrences int
	Secrets          []string
}

type Pipeline struct {
	MaxEvidence     int
	MaxSummaryBytes int
}

func (p Pipeline) Process(in Input) domain.StructuredFeedback {
	redacted := NewRedactor(in.Secrets).Redact(observationText(in))
	humanText := normalizeHumanText(redacted)
	category, diagnostics := classify(in, humanText)
	maxEvidence := p.maxEvidence()
	var evidence []domain.Evidence
	var evidenceTruncated bool
	if len(diagnostics) >= maxEvidence {
		evidence = append(evidence, diagnostics[:maxEvidence]...)
		evidenceTruncated = len(diagnostics) > maxEvidence || len(meaningfulLines(humanText)) > 0
	} else {
		evidence, evidenceTruncated = compressEvidence(humanText, maxEvidence-len(diagnostics))
		evidence = append(diagnostics, evidence...)
	}
	summary, summaryTruncated := summarize(category, humanText, p.maxSummaryBytes())
	return domain.StructuredFeedback{
		Category:            category,
		StageID:             in.StageID,
		Summary:             summary,
		Fingerprint:         Fingerprint(in.StageID, category, redacted),
		Evidence:            evidence,
		Retryable:           retryable(category),
		OutputTruncated:     in.Observation.OutputTruncated || evidenceTruncated || summaryTruncated,
		PreviousOccurrences: in.PriorOccurrences,
	}
}

func (p Pipeline) maxEvidence() int {
	if p.MaxEvidence > 0 {
		return p.MaxEvidence
	}
	return 6
}

func (p Pipeline) maxSummaryBytes() int {
	if p.MaxSummaryBytes > 0 {
		return p.MaxSummaryBytes
	}
	return 240
}

func classify(in Input, text string) (string, []domain.Evidence) {
	code := in.Code
	if code == "" {
		code = in.Observation.Code
	}
	compiledRules, diagnostics := compileClassifierRules(in.Rules)
	upperCode := strings.ToUpper(code)
	switch upperCode {
	case "INVALID_JSON", "INVALID_AGENT_JSON", "INVALID_DECISION_JSON", "INVALID_ACTION_ARGS":
		return "PROTOCOL_ERROR", diagnostics
	}
	switch upperCode {
	case "APPROVAL_REQUIRED":
		return "APPROVAL_REQUIRED", diagnostics
	case "UNKNOWN_ACTION", "REPOSITORY_ESCAPE", "PATH_DENIED", "SYMLINK_ESCAPE", "CREDENTIAL_ACCESS", "GIT_INTERNALS", "PROTECTED_PATH", "UNCONFIGURED_CHECK", "RAW_SHELL_DENIED":
		return "POLICY_DENIED", diagnostics
	}
	switch upperCode {
	case "STALE_PATCH":
		return "STALE_PATCH", diagnostics
	case "EMPTY_PATCH":
		return "EMPTY_PATCH", diagnostics
	case "PATCH_CONFLICT", "INVALID_PATCH_SCHEMA":
		return "PATCH_FAILURE", diagnostics
	case "REGRESSION":
		return "REGRESSION", diagnostics
	}
	switch upperCode {
	case executor.CodeTimeout:
		return "TIMEOUT", diagnostics
	case executor.CodeCancelled, "USER_CANCELLED":
		return "CANCELLED", diagnostics
	case executor.CodeStartError:
		if missingExecutable(text) {
			return "MISSING_EXECUTABLE", diagnostics
		}
		return "ENVIRONMENT_FAILURE", diagnostics
	case executor.CodeExecutionError:
		return "ENVIRONMENT_FAILURE", diagnostics
	case executor.CodeCleanupError:
		return "INCOMPLETE_PROCESS_CLEANUP", diagnostics
	}

	for _, rule := range compiledRules {
		if rule.re.MatchString(text) {
			return rule.category, diagnostics
		}
	}

	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "syntax error") || strings.Contains(lower, "compile error") || strings.Contains(lower, "compilation failed"):
		return "COMPILE_FAILURE", diagnostics
	case strings.Contains(lower, "cannot use ") || strings.Contains(lower, "undefined:") || strings.Contains(lower, "type error") || strings.Contains(lower, "mismatched types"):
		return "TYPE_FAILURE", diagnostics
	case strings.Contains(lower, "lint") || strings.Contains(lower, "gofmt") || strings.Contains(lower, "go vet") || strings.Contains(lower, "eslint"):
		return "LINT_FAILURE", diagnostics
	case strings.Contains(lower, "build failed") || strings.Contains(lower, "build failure"):
		return "BUILD_FAILURE", diagnostics
	case strings.Contains(lower, "--- fail:") || strings.Contains(lower, "fail ") || strings.Contains(lower, " expected "):
		return "TEST_FAILURE", diagnostics
	}
	if in.Observation.ExitCode != nil && *in.Observation.ExitCode != 0 {
		return "VALIDATION_FAILED", diagnostics
	}
	if in.PriorOccurrences > 0 {
		return "NO_PROGRESS", diagnostics
	}
	return "PROGRESS", diagnostics
}

type compiledClassifierRule struct {
	category string
	re       *regexp.Regexp
}

type cachedClassifierPattern struct {
	re    *regexp.Regexp
	valid bool
}

const classifierPatternCacheLimit = 128

type classifierRegexpCache struct {
	mu      sync.RWMutex
	limit   int
	entries map[string]cachedClassifierPattern
}

var classifierPatternCache = newClassifierPatternCache(classifierPatternCacheLimit)

func compileClassifierRules(rules []config.ClassifierRule) ([]compiledClassifierRule, []domain.Evidence) {
	compiled := make([]compiledClassifierRule, 0, len(rules))
	var diagnostics []domain.Evidence
	for index, rule := range rules {
		if rule.Category == "" || rule.Pattern == "" {
			continue
		}
		pattern := cachedClassifierRegexp(rule.Pattern)
		if !pattern.valid {
			diagnostics = append(diagnostics, domain.Evidence{
				Source:  "classifier",
				Message: fmt.Sprintf("invalid classifier rule at index %d", index),
			})
			continue
		}
		compiled = append(compiled, compiledClassifierRule{category: rule.Category, re: pattern.re})
	}
	return compiled, diagnostics
}

func cachedClassifierRegexp(pattern string) cachedClassifierPattern {
	return classifierPatternCache.loadOrStore(pattern)
}

func newClassifierPatternCache(limit int) *classifierRegexpCache {
	return &classifierRegexpCache{
		limit:   limit,
		entries: make(map[string]cachedClassifierPattern),
	}
}

func (c *classifierRegexpCache) loadOrStore(pattern string) cachedClassifierPattern {
	c.mu.RLock()
	cached, ok := c.entries[pattern]
	c.mu.RUnlock()
	if ok {
		return cached
	}

	compiled, err := regexp.Compile(pattern)
	value := cachedClassifierPattern{re: compiled, valid: err == nil}

	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.entries[pattern]; ok {
		return cached
	}
	if len(c.entries) < c.limit {
		c.entries[pattern] = value
	}
	return value
}

func (c *classifierRegexpCache) len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func missingExecutable(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "executable file not found") ||
		strings.Contains(lower, "file not found") ||
		strings.Contains(lower, "no such file or directory")
}

func retryable(category string) bool {
	switch category {
	case "APPROVAL_REQUIRED", "POLICY_DENIED", "PROTOCOL_ERROR", "INCOMPLETE_PROCESS_CLEANUP":
		return false
	default:
		return true
	}
}
