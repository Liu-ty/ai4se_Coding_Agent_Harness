// Package workspace provides repository-bound filesystem helpers.
package workspace

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrOutsideRoot   = errors.New("path is outside workspace root")
	ErrProtectedPath = errors.New("path is protected")
)

// Resolve returns the canonical path for a workspace-relative path.
func Resolve(root, relative string) (string, error) {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	if isAbsolute(relative) {
		return "", ErrOutsideRoot
	}
	relative = filepath.Clean(relative)
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", ErrOutsideRoot
	}
	if protected(relative) {
		return "", ErrProtectedPath
	}

	candidate := filepath.Join(canonicalRoot, relative)
	canonicalCandidate, err := resolveExistingParent(candidate)
	if err != nil {
		return "", err
	}
	inside, err := within(canonicalRoot, canonicalCandidate)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", ErrOutsideRoot
	}
	resolvedRel, err := filepath.Rel(canonicalRoot, canonicalCandidate)
	if err != nil {
		return "", fmt.Errorf("compare workspace path: %w", err)
	}
	if protected(resolvedRel) {
		return "", ErrProtectedPath
	}
	return canonicalCandidate, nil
}

func resolveExistingParent(path string) (string, error) {
	missing := make([]string, 0)
	current := path
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func within(root, candidate string) (bool, error) {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false, fmt.Errorf("compare workspace path: %w", err)
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func isAbsolute(path string) bool {
	path = strings.ReplaceAll(path, "\\", "/")
	return filepath.IsAbs(path) || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") ||
		(len(path) >= 2 && path[1] == ':')
}

func protected(path string) bool {
	clean := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	for _, part := range strings.Split(clean, "/") {
		if part == ".git" || strings.HasPrefix(part, ".env") || strings.Contains(part, "secret") || strings.Contains(part, "credential") || strings.Contains(part, "vault") {
			return true
		}
	}
	return false
}
