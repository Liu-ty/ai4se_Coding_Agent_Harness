package tools

import (
	"fmt"
	"strconv"
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
	seen := make(map[string]struct{})
	result := patchHeaders{}
	var oldPath string
	currentHasHunk := false
	inHunk, oldRemaining, newRemaining := false, 0, 0

	for _, line := range strings.Split(patch, "\n") {
		if inHunk {
			changed, err := consumeHunkLine(line, &oldRemaining, &newRemaining)
			if err != nil {
				return patchHeaders{}, err
			}
			result.changedLines += changed
			if result.changedLines > limits.MaxChangedLines {
				return patchHeaders{}, ErrPatchLimit
			}
			inHunk = oldRemaining != 0 || newRemaining != 0
			continue
		}

		switch {
		case strings.HasPrefix(line, "rename from "), strings.HasPrefix(line, "rename to "), strings.HasPrefix(line, "similarity index "):
			return patchHeaders{}, ErrInvalidPatch
		case strings.HasPrefix(line, "--- "):
			if oldPath != "" || (len(result.paths) > 0 && !currentHasHunk) {
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
			if len(result.paths) > limits.MaxFiles {
				return patchHeaders{}, ErrPatchLimit
			}
			oldPath, currentHasHunk = "", false
		case strings.HasPrefix(line, "@@ "):
			if len(result.paths) == 0 || oldPath != "" {
				return patchHeaders{}, ErrInvalidPatch
			}
			var err error
			oldRemaining, newRemaining, err = parseHunkHeader(line)
			if err != nil {
				return patchHeaders{}, err
			}
			currentHasHunk = true
			inHunk = oldRemaining != 0 || newRemaining != 0
		}
	}
	if len(result.paths) == 0 || oldPath != "" || !currentHasHunk || inHunk {
		return patchHeaders{}, ErrInvalidPatch
	}
	return result, nil
}

func consumeHunkLine(line string, oldRemaining, newRemaining *int) (int, error) {
	if strings.HasPrefix(line, "\\ No newline at end of file") {
		return 0, nil
	}
	switch {
	case strings.HasPrefix(line, "+"):
		if *newRemaining == 0 {
			return 0, ErrInvalidPatch
		}
		*newRemaining--
		return 1, nil
	case strings.HasPrefix(line, "-"):
		if *oldRemaining == 0 {
			return 0, ErrInvalidPatch
		}
		*oldRemaining--
		return 1, nil
	case strings.HasPrefix(line, " "):
		if *oldRemaining == 0 || *newRemaining == 0 {
			return 0, ErrInvalidPatch
		}
		*oldRemaining--
		*newRemaining--
		return 0, nil
	default:
		return 0, ErrInvalidPatch
	}
}

func parseHunkHeader(line string) (int, int, error) {
	fields := strings.Fields(line)
	if len(fields) < 4 || fields[0] != "@@" || fields[3] != "@@" {
		return 0, 0, ErrInvalidPatch
	}
	oldCount, err := hunkCount(fields[1], "-")
	if err != nil {
		return 0, 0, err
	}
	newCount, err := hunkCount(fields[2], "+")
	if err != nil {
		return 0, 0, err
	}
	return oldCount, newCount, nil
}

func hunkCount(value, prefix string) (int, error) {
	if !strings.HasPrefix(value, prefix) {
		return 0, ErrInvalidPatch
	}
	parts := strings.Split(strings.TrimPrefix(value, prefix), ",")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidPatch
	}
	if len(parts) == 1 {
		return 1, nil
	}
	count, err := strconv.Atoi(parts[1])
	if err != nil || count < 0 {
		return 0, fmt.Errorf("%w: invalid hunk count", ErrInvalidPatch)
	}
	return count, nil
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
