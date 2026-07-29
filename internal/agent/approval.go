package agent

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
)

// ApprovalRequired is the stable, display-safe request published by the
// production governance loop. Digest continues to bind the exact unredacted
// action and evidence; Action is the canonical redacted representation shown
// to an operator.
type ApprovalRequired struct {
	Digest           policy.ApprovalDigest     `json:"digest"`
	Action           domain.Action             `json:"action"`
	AffectedFiles    []string                  `json:"affected_files"`
	Risk             policy.Risk               `json:"risk"`
	RiskReason       string                    `json:"risk_reason"`
	BaselineEvidence []ApprovalEvidenceBinding `json:"baseline_evidence"`
}

type ApprovalEvidenceBinding struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

func newApprovalRequired(
	profile domain.PermissionProfile,
	action domain.Action,
	decision policy.Decision,
	baselines map[string]string,
) ApprovalRequired {
	redactor := feedback.NewRedactor(nil)
	canonical := canonicalRedactedAction(action, redactor)
	evidence := make([]ApprovalEvidenceBinding, 0, len(baselines))
	for name, digest := range baselines {
		evidence = append(evidence, ApprovalEvidenceBinding{
			Name: redactor.Redact(name), Digest: redactor.Redact(digest),
		})
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].Name < evidence[j].Name })
	reason := decision.Message
	if reason == "" {
		reason = string(profile) + " profile requires approval for " + action.Kind
	}
	files := affectedFiles(action)
	for index, path := range files {
		files[index] = redactor.Redact(path)
	}
	return ApprovalRequired{
		Digest:           decision.Digest,
		Action:           canonical,
		AffectedFiles:    files,
		Risk:             decision.Risk,
		RiskReason:       redactor.Redact(reason),
		BaselineEvidence: evidence,
	}
}

func canonicalRedactedAction(action domain.Action, redactor feedback.Redactor) domain.Action {
	action = policy.CanonicalAction(action)
	var value any
	if json.Unmarshal(action.Args, &value) != nil {
		return domain.Action{Kind: action.Kind, Args: json.RawMessage(`{}`)}
	}
	value = redactJSONValue(value, redactor)
	raw, err := json.Marshal(value)
	if err != nil {
		return domain.Action{Kind: action.Kind, Args: json.RawMessage(`{}`)}
	}
	return domain.Action{Kind: action.Kind, Args: raw}
}

func redactJSONValue(value any, redactor feedback.Redactor) any {
	switch typed := value.(type) {
	case string:
		return redactor.Redact(typed)
	case []any:
		for index, nested := range typed {
			typed[index] = redactJSONValue(nested, redactor)
		}
	case map[string]any:
		for key, nested := range typed {
			typed[key] = redactJSONValue(nested, redactor)
		}
	}
	return value
}

func affectedFiles(action domain.Action) []string {
	var args struct {
		Path  string `json:"path"`
		Patch string `json:"patch"`
	}
	if json.Unmarshal(action.Args, &args) != nil {
		return []string{}
	}
	files := make(map[string]struct{})
	if action.Kind == "create_file" && args.Path != "" {
		files[args.Path] = struct{}{}
	}
	if action.Kind == "apply_patch" {
		for _, line := range strings.Split(args.Patch, "\n") {
			if !strings.HasPrefix(line, "+++ ") {
				continue
			}
			fields := strings.Fields(strings.TrimPrefix(line, "+++ "))
			if len(fields) == 0 || fields[0] == "/dev/null" {
				continue
			}
			files[strings.TrimPrefix(fields[0], "b/")] = struct{}{}
		}
	}
	result := make([]string, 0, len(files))
	for path := range files {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}
