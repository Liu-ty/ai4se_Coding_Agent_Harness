package tools

import (
	"fmt"
	"strings"
)

type patchHeaders struct {
	paths        []string
	changedLines int
}

func parsePatchHeaders(patch string, limits PatchLimits) (patchHeaders, error) {
	if patch == "" || strings.Contains(patch, "\x00") || strings.Contains(patch, "GIT binary patch") || strings.Contains(patch, "Binary files ") {
		return patchHeaders{}, ErrInvalidPatch
	}
	limits = boundedPatchLimits(limits)
	lines := strings.Split(patch, "\n")
	seen := make(map[string]struct{})
	result := patchHeaders{}
	var oldPath string
	var hunks []bool
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "rename from "), strings.HasPrefix(line, "rename to "), strings.HasPrefix(line, "similarity index "):
			return patchHeaders{}, ErrInvalidPatch
		case strings.HasPrefix(line, "--- "):
			if oldPath != "" || (len(hunks) > 0 && !hunks[len(hunks)-1]) {
				return patchHeaders{}, ErrInvalidPatch
			}
			path, err := patchPath(line[4:], "a/")
			if err != nil {
				return patchHeaders{}, err
			}
			oldPath = path
		case strings.HasPrefix(line, "+++ "):
			if oldPath == "" {
				return patchHeaders{}, ErrInvalidPatch
			}
			path, err := patchPath(line[4:], "b/")
			if err != nil || path != oldPath {
				return patchHeaders{}, ErrInvalidPatch
			}
			if _, duplicate := seen[path]; duplicate {
				return patchHeaders{}, ErrInvalidPatch
			}
			seen[path] = struct{}{}
			result.paths = append(result.paths, path)
			hunks = append(hunks, false)
			oldPath = ""
		case strings.HasPrefix(line, "@@ "):
			if len(result.paths) == 0 || oldPath != "" {
				return patchHeaders{}, ErrInvalidPatch
			}
			hunks[len(hunks)-1] = true
		case strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-"):
			if len(result.paths) > 0 && oldPath == "" && hunks[len(hunks)-1] {
				result.changedLines++
			}
		}
		if len(result.paths) > limits.MaxFiles || result.changedLines > limits.MaxChangedLines {
			return patchHeaders{}, ErrPatchLimit
		}
	}
	if len(result.paths) == 0 || oldPath != "" || !hunks[len(hunks)-1] {
		return patchHeaders{}, ErrInvalidPatch
	}
	return result, nil
}

func patchPath(header, prefix string) (string, error) {
	path := strings.Fields(header)
	if len(path) == 0 || path[0] == "/dev/null" || !strings.HasPrefix(path[0], prefix) {
		return "", ErrInvalidPatch
	}
	value := strings.TrimPrefix(path[0], prefix)
	if value == "" {
		return "", fmt.Errorf("%w: empty path", ErrInvalidPatch)
	}
	return value, nil
}
