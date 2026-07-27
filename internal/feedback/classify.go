package feedback

import (
	"regexp"
	"strings"

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
	category := classify(in, humanText)
	evidence, evidenceTruncated := compressEvidence(humanText, p.maxEvidence())
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

func classify(in Input, text string) string {
	code := in.Code
	if code == "" {
		code = in.Observation.Code
	}
	upperCode := strings.ToUpper(code)
	switch upperCode {
	case "INVALID_JSON", "INVALID_AGENT_JSON", "INVALID_DECISION_JSON", "INVALID_ACTION_ARGS":
		return "PROTOCOL_ERROR"
	}
	switch upperCode {
	case "APPROVAL_REQUIRED":
		return "APPROVAL_REQUIRED"
	case "UNKNOWN_ACTION", "REPOSITORY_ESCAPE", "PATH_DENIED", "SYMLINK_ESCAPE", "CREDENTIAL_ACCESS", "GIT_INTERNALS", "PROTECTED_PATH", "UNCONFIGURED_CHECK", "RAW_SHELL_DENIED":
		return "POLICY_DENIED"
	}
	switch upperCode {
	case "STALE_PATCH":
		return "STALE_PATCH"
	case "EMPTY_PATCH":
		return "EMPTY_PATCH"
	case "PATCH_CONFLICT", "INVALID_PATCH_SCHEMA":
		return "PATCH_FAILURE"
	case "REGRESSION":
		return "REGRESSION"
	}
	switch upperCode {
	case executor.CodeTimeout:
		return "TIMEOUT"
	case executor.CodeCancelled, "USER_CANCELLED":
		return "CANCELLED"
	case executor.CodeStartError:
		if missingExecutable(text) {
			return "MISSING_EXECUTABLE"
		}
		return "ENVIRONMENT_FAILURE"
	}

	for _, rule := range in.Rules {
		if rule.Category == "" || rule.Pattern == "" {
			continue
		}
		re, err := regexp.Compile(rule.Pattern)
		if err == nil && re.MatchString(text) {
			return rule.Category
		}
	}

	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "syntax error") || strings.Contains(lower, "compile error") || strings.Contains(lower, "compilation failed"):
		return "COMPILE_FAILURE"
	case strings.Contains(lower, "cannot use ") || strings.Contains(lower, "undefined:") || strings.Contains(lower, "type error") || strings.Contains(lower, "mismatched types"):
		return "TYPE_FAILURE"
	case strings.Contains(lower, "lint") || strings.Contains(lower, "gofmt") || strings.Contains(lower, "go vet") || strings.Contains(lower, "eslint"):
		return "LINT_FAILURE"
	case strings.Contains(lower, "build failed") || strings.Contains(lower, "build failure"):
		return "BUILD_FAILURE"
	case strings.Contains(lower, "--- fail:") || strings.Contains(lower, "fail ") || strings.Contains(lower, " expected "):
		return "TEST_FAILURE"
	}
	if in.Observation.ExitCode != nil && *in.Observation.ExitCode != 0 {
		return "VALIDATION_FAILED"
	}
	if in.PriorOccurrences > 0 {
		return "NO_PROGRESS"
	}
	return "PROGRESS"
}

func missingExecutable(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "executable file not found") ||
		strings.Contains(lower, "file not found") ||
		strings.Contains(lower, "no such file or directory")
}

func retryable(category string) bool {
	switch category {
	case "APPROVAL_REQUIRED", "POLICY_DENIED", "PROTOCOL_ERROR":
		return false
	default:
		return true
	}
}
