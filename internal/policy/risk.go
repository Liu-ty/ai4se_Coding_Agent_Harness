package policy

import (
	"encoding/json"
	pathpkg "path"
	"strings"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

type Risk string

const (
	RiskNormal     Risk = "NORMAL"
	RiskGuarded    Risk = "GUARDED"
	RiskHardDenied Risk = "HARD_DENIED"
)

const (
	maxFiles        = 5
	maxChangedLines = 500
	maxFileBytes    = 1 << 20
)

var canonicalActions = map[string]bool{
	"list_files": true, "search_text": true, "read_file": true,
	"apply_patch": true, "create_file": true, "run_check": true, "finish": true,
}

func classifyRisk(ctx Context, action domain.Action) (Risk, string, string) {
	if !canonicalActions[action.Kind] {
		return RiskHardDenied, "UNKNOWN_ACTION", "action is not in the course-delivery registry"
	}

	var args any
	if len(action.Args) == 0 || json.Unmarshal(action.Args, &args) != nil {
		return RiskHardDenied, "INVALID_ACTION_ARGS", "action arguments must be valid JSON"
	}
	facts := inspectArgs(args)
	if facts.hardDenied {
		return RiskHardDenied, facts.code, facts.message
	}
	if risk, code, message := validateCanonicalAction(ctx, action.Kind, action.Args, &facts); risk != RiskNormal {
		return risk, code, message
	}
	if !isMutation(action.Kind) {
		return RiskNormal, "", ""
	}
	if facts.guarded {
		return RiskGuarded, facts.code, facts.message
	}
	if ctx.Dirty {
		return RiskGuarded, "DIRTY_WORKTREE", "mutation targets a dirty worktree"
	}
	return RiskNormal, "", ""
}

func validateCanonicalAction(ctx Context, kind string, raw json.RawMessage, facts *actionFacts) (Risk, string, string) {
	switch kind {
	case "run_check":
		return validateRunCheck(ctx, raw)
	case "create_file":
		return validateCreateFile(raw, facts)
	case "apply_patch":
		return validatePatch(raw, facts)
	default:
		return RiskNormal, "", ""
	}
}

func validateRunCheck(ctx Context, raw json.RawMessage) (Risk, string, string) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != 1 || fields["id"] == nil {
		return RiskHardDenied, "INVALID_CHECK_SCHEMA", "run_check requires one configured check identifier"
	}
	var id string
	if json.Unmarshal(fields["id"], &id) != nil || id == "" {
		return RiskHardDenied, "INVALID_CHECK_SCHEMA", "run_check requires one configured check identifier"
	}
	for _, configured := range ctx.ConfiguredChecks {
		if id == configured {
			return RiskNormal, "", ""
		}
	}
	return RiskHardDenied, "UNCONFIGURED_CHECK", "run_check identifier is not configured"
}

func validateCreateFile(raw json.RawMessage, facts *actionFacts) (Risk, string, string) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields["path"] == nil || fields["content"] == nil {
		return RiskHardDenied, "INVALID_CREATE_SCHEMA", "create_file requires a path and UTF-8 content"
	}
	var path string
	var rawContent any
	if json.Unmarshal(fields["path"], &path) != nil || json.Unmarshal(fields["content"], &rawContent) != nil || path == "" {
		return RiskHardDenied, "INVALID_CREATE_SCHEMA", "create_file requires a path and UTF-8 content"
	}
	content, ok := rawContent.(string)
	if !ok {
		return RiskHardDenied, "INVALID_CREATE_SCHEMA", "create_file requires a path and UTF-8 content"
	}
	inspectPath(path, facts)
	if facts.hardDenied {
		return RiskHardDenied, facts.code, facts.message
	}
	if len(content) > maxFileBytes {
		facts.guarded, facts.code, facts.message = true, "FILE_TOO_LARGE", "mutation writes too many bytes"
	}
	return RiskNormal, "", ""
}

func validatePatch(raw json.RawMessage, facts *actionFacts) (Risk, string, string) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields["patch"] == nil {
		return RiskHardDenied, "INVALID_PATCH_SCHEMA", "apply_patch requires a patch payload"
	}
	var patch string
	if json.Unmarshal(fields["patch"], &patch) != nil || patch == "" {
		return RiskHardDenied, "INVALID_PATCH_SCHEMA", "apply_patch requires a patch payload"
	}

	files := make(map[string]struct{})
	changedLines := 0
	var oldPath string
	pairs := 0
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "--- ") {
			oldPath = diffPath(line[4:], "a/")
			if oldPath == "" {
				return RiskHardDenied, "INVALID_PATCH_SCHEMA", "apply_patch requires unified diff file headers"
			}
			continue
		}
		if strings.HasPrefix(line, "+++ ") {
			newPath := diffPath(line[4:], "b/")
			if oldPath == "" || newPath == "" {
				return RiskHardDenied, "INVALID_PATCH_SCHEMA", "apply_patch requires unified diff file headers"
			}
			for _, path := range []string{oldPath, newPath} {
				if path == "/dev/null" {
					continue
				}
				inspectPath(path, facts)
				files[path] = struct{}{}
			}
			pairs++
			oldPath = ""
			continue
		}
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			changedLines++
		}
	}
	if pairs == 0 || oldPath != "" || len(files) == 0 {
		return RiskHardDenied, "INVALID_PATCH_SCHEMA", "apply_patch requires unified diff file headers"
	}
	if facts.hardDenied {
		return RiskHardDenied, facts.code, facts.message
	}
	if len(files) > maxFiles {
		facts.guarded, facts.code, facts.message = true, "TOO_MANY_FILES", "mutation affects too many files"
	}
	if changedLines > maxChangedLines {
		facts.guarded, facts.code, facts.message = true, "TOO_MANY_CHANGED_LINES", "mutation changes too many lines"
	}
	return RiskNormal, "", ""
}

func diffPath(header, prefix string) string {
	fields := strings.Fields(header)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimPrefix(fields[0], prefix)
}

type actionFacts struct {
	hardDenied bool
	guarded    bool
	code       string
	message    string
}

func inspectArgs(args any) actionFacts {
	var facts actionFacts
	inspectValue(args, "", &facts)
	return facts
}

func inspectValue(value any, key string, facts *actionFacts) {
	if facts.hardDenied {
		return
	}
	switch v := value.(type) {
	case map[string]any:
		for k, nested := range v {
			lower := strings.ToLower(k)
			if truthyHardDeny(lower, nested) {
				facts.hardDenied, facts.code, facts.message = true, "UNSAFE_ACTION_FACT", "action includes a prohibited workspace risk"
				return
			}
			if truthyGuard(lower, nested) {
				facts.guarded, facts.code, facts.message = true, "GUARDED_MUTATION", "action exceeds normal mutation limits"
			}
			inspectValue(nested, lower, facts)
		}
	case []any:
		for _, nested := range v {
			inspectValue(nested, key, facts)
		}
	case string:
		if isPathKey(key) {
			inspectPath(v, facts)
		}
	}
}

func truthyHardDeny(key string, value any) bool {
	b, ok := value.(bool)
	if !ok || !b {
		return false
	}
	return key == "repository_escape" || key == "symlink_escape" || key == "binary" ||
		key == "credential_access" || key == "network" || key == "raw_shell" || key == "key_endpoint_mismatch"
}

func truthyGuard(key string, value any) bool {
	b, ok := value.(bool)
	return ok && b && (key == "protected" || key == "large")
}

func isPathKey(key string) bool {
	return key == "path" || key == "paths" || key == "file" || key == "files"
}

func inspectPath(rawPath string, facts *actionFacts) {
	clean := strings.ReplaceAll(rawPath, "\\", "/")
	if strings.HasPrefix(clean, "/") || (len(clean) >= 2 && clean[1] == ':') {
		facts.hardDenied, facts.code, facts.message = true, "REPOSITORY_ESCAPE", "action path escapes the repository"
		return
	}
	clean = pathpkg.Clean(clean)
	lower := strings.ToLower(clean)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		facts.hardDenied, facts.code, facts.message = true, "REPOSITORY_ESCAPE", "action path escapes the repository"
		return
	}
	if lower == ".git" || strings.HasPrefix(lower, ".git/") {
		facts.hardDenied, facts.code, facts.message = true, "GIT_INTERNALS", "action targets repository internals"
		return
	}
	if strings.Contains(lower, "credential") || strings.Contains(lower, "secret") || strings.Contains(lower, "vault") || strings.HasSuffix(lower, ".env") {
		facts.hardDenied, facts.code, facts.message = true, "CREDENTIAL_ACCESS", "action targets credential material"
		return
	}
	if strings.HasPrefix(lower, ".github/") || lower == "go.mod" || lower == "go.sum" || strings.HasPrefix(lower, "vendor/") {
		facts.guarded, facts.code, facts.message = true, "PROTECTED_PATH", "action targets a protected path"
	}
}
