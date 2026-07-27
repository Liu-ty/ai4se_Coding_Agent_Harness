package policy

import (
	"encoding/json"
	"path/filepath"
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
	if len(action.Args) > 0 && json.Unmarshal(action.Args, &args) != nil {
		return RiskHardDenied, "INVALID_ACTION_ARGS", "action arguments must be valid JSON"
	}
	facts := inspectArgs(args)
	if facts.hardDenied {
		return RiskHardDenied, facts.code, facts.message
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
		if (key == "paths" || key == "files") && len(v) > maxFiles {
			facts.guarded, facts.code, facts.message = true, "TOO_MANY_FILES", "mutation affects too many files"
		}
		for _, nested := range v {
			inspectValue(nested, key, facts)
		}
	case string:
		if isPathKey(key) {
			inspectPath(v, facts)
		}
	case float64:
		if (key == "files" || key == "file_count") && v > maxFiles {
			facts.guarded, facts.code, facts.message = true, "TOO_MANY_FILES", "mutation affects too many files"
		}
		if (key == "changed_lines" || key == "line_count") && v > maxChangedLines {
			facts.guarded, facts.code, facts.message = true, "TOO_MANY_CHANGED_LINES", "mutation changes too many lines"
		}
		if (key == "bytes" || key == "size_bytes" || key == "file_bytes") && v > maxFileBytes {
			facts.guarded, facts.code, facts.message = true, "FILE_TOO_LARGE", "mutation writes too many bytes"
		}
	}
}

func truthyHardDeny(key string, value any) bool {
	b, ok := value.(bool)
	if !ok || !b {
		return false
	}
	return key == "repository_escape" || key == "symlink_escape" || key == "binary" ||
		key == "credential_access" || key == "network" || key == "raw_shell"
}

func truthyGuard(key string, value any) bool {
	b, ok := value.(bool)
	return ok && b && (key == "protected" || key == "large")
}

func isPathKey(key string) bool {
	return key == "path" || key == "paths" || key == "file" || key == "files"
}

func inspectPath(path string, facts *actionFacts) {
	clean := strings.ReplaceAll(path, "\\", "/")
	lower := strings.ToLower(clean)
	if filepath.IsAbs(path) || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") ||
		(len(clean) >= 2 && clean[1] == ':') {
		facts.hardDenied, facts.code, facts.message = true, "REPOSITORY_ESCAPE", "action path escapes the repository"
		return
	}
	if strings.HasPrefix(lower, ".git/") || lower == ".git" {
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
