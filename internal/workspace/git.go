package workspace

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitBaseline returns the current repository revision and porcelain status.
func GitBaseline(ctx context.Context, root string) (map[string]string, error) {
	root, err := ResolveRoot(root)
	if err != nil {
		return nil, err
	}
	head, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	status, err := gitOutput(ctx, root, "status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}
	return map[string]string{"head": head, "status": status}, nil
}

// ResolveRoot returns the canonical repository root without applying path rules.
func ResolveRoot(root string) (string, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	return resolved, nil
}

func gitOutput(ctx context.Context, root string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}
