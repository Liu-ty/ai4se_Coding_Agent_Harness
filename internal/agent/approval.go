package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

const (
	maxApprovalPreviewBytes = 2048
	maxApprovalTextBytes    = 512
	maxApprovalEvidence     = 16
	maxApprovalFiles        = 16
)

type approvalContentDisplay struct {
	SHA256    string `json:"sha256"`
	Preview   string `json:"preview"`
	Truncated bool   `json:"truncated"`
}

func newApprovalRequired(
	profile domain.PermissionProfile,
	action domain.Action,
	decision policy.Decision,
	baselines map[string]string,
	redactor feedback.Redactor,
) ApprovalRequired {
	displayAction := boundedRedactedAction(action, redactor)
	evidence := make([]ApprovalEvidenceBinding, 0, min(len(baselines), maxApprovalEvidence))
	for name, digest := range baselines {
		evidence = append(evidence, ApprovalEvidenceBinding{
			Name:   boundedRedacted(name, maxApprovalTextBytes, redactor),
			Digest: boundedRedacted(digest, maxApprovalTextBytes, redactor),
		})
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].Name < evidence[j].Name })
	if len(evidence) > maxApprovalEvidence {
		evidence = evidence[:maxApprovalEvidence]
	}
	reason := decision.Message
	if reason == "" {
		reason = string(profile) + " profile requires approval for " + action.Kind
	}
	files := affectedFiles(action)
	if len(files) > maxApprovalFiles {
		files = files[:maxApprovalFiles]
	}
	for index, path := range files {
		files[index] = boundedRedacted(path, maxApprovalTextBytes, redactor)
	}
	return ApprovalRequired{
		Digest:           decision.Digest,
		Action:           displayAction,
		AffectedFiles:    files,
		Risk:             decision.Risk,
		RiskReason:       boundedRedacted(reason, maxApprovalTextBytes, redactor),
		BaselineEvidence: evidence,
	}
}

func boundedRedactedAction(action domain.Action, redactor feedback.Redactor) domain.Action {
	action = policy.CanonicalAction(action)
	var args map[string]json.RawMessage
	if json.Unmarshal(action.Args, &args) != nil {
		return summarizedAction(action, redactor)
	}
	var display any
	switch action.Kind {
	case "apply_patch":
		var patch string
		if json.Unmarshal(args["patch"], &patch) != nil {
			return summarizedAction(action, redactor)
		}
		display = map[string]any{"patch": patchDisplay(patch, redactor)}
	case "create_file":
		var path, content string
		if json.Unmarshal(args["path"], &path) != nil ||
			json.Unmarshal(args["content"], &content) != nil {
			return summarizedAction(action, redactor)
		}
		display = map[string]any{
			"path":    boundedRedacted(path, maxApprovalTextBytes, redactor),
			"content": contentDisplay(content, redactor),
		}
	case "run_check":
		var id string
		if json.Unmarshal(args["id"], &id) != nil {
			return summarizedAction(action, redactor)
		}
		display = map[string]string{"id": boundedRedacted(id, maxApprovalTextBytes, redactor)}
	default:
		return summarizedAction(action, redactor)
	}
	raw, err := json.Marshal(display)
	if err != nil {
		return domain.Action{Kind: action.Kind, Args: json.RawMessage(`{}`)}
	}
	return domain.Action{Kind: action.Kind, Args: raw}
}

func summarizedAction(action domain.Action, redactor feedback.Redactor) domain.Action {
	raw, err := json.Marshal(map[string]any{
		"summary": contentDisplay(string(action.Args), redactor),
	})
	if err != nil {
		raw = json.RawMessage(`{}`)
	}
	return domain.Action{Kind: action.Kind, Args: raw}
}

func contentDisplay(value string, redactor feedback.Redactor) approvalContentDisplay {
	sum := sha256.Sum256([]byte(value))
	redacted := redactor.Redact(value)
	preview := bounded(redacted, maxApprovalPreviewBytes)
	return approvalContentDisplay{
		SHA256:    fmt.Sprintf("%x", sum[:]),
		Preview:   preview,
		Truncated: len(redacted) > len(preview),
	}
}

func patchDisplay(patch string, redactor feedback.Redactor) approvalContentDisplay {
	hunks := make([]string, 0)
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "@@ ") {
			hunks = append(hunks, line)
		}
	}
	sum := sha256.Sum256([]byte(patch))
	redacted := redactor.Redact(strings.Join(hunks, "\n"))
	preview := bounded(redacted, maxApprovalPreviewBytes)
	return approvalContentDisplay{
		SHA256:    fmt.Sprintf("%x", sum[:]),
		Preview:   preview,
		Truncated: preview != redactor.Redact(patch),
	}
}

func boundedRedacted(value string, limit int, redactor feedback.Redactor) string {
	return bounded(redactor.Redact(value), limit)
}

func bounded(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
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
