package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/workspace"
)

const (
	maximumPatchFiles = 5
	maximumPatchLines = 500
)

type PatchLimits struct {
	MaxFiles        int
	MaxChangedLines int
}

type gitRunner interface {
	run(context.Context, string, []string, []byte) ([]byte, error)
}

type installedGit struct{}

func (installedGit) run(ctx context.Context, root string, args []string, input []byte) ([]byte, error) {
	return runGit(ctx, root, args, input)
}

func runGit(ctx context.Context, root string, args []string, input []byte) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = root
	command.Stdin = bytes.NewReader(input)
	return command.CombinedOutput()
}

type patchTool struct {
	root   string
	limits PatchLimits
	git    gitRunner
}

func NewPatchTool(root string, limits PatchLimits) Tool {
	return newPatchTool(root, limits, installedGit{})
}

func newPatchTool(root string, limits PatchLimits, git gitRunner) patchTool {
	return patchTool{root: root, limits: boundedPatchLimits(limits), git: git}
}

func (patchTool) Kind() string { return "apply_patch" }

func (t patchTool) Execute(ctx context.Context, raw json.RawMessage) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	var args struct {
		Patch     string            `json:"patch"`
		Baselines map[string]string `json:"baselines"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.Patch == "" {
		return Result{}, ErrInvalidArgs
	}
	headers, err := parsePatchHeaders(args.Patch, t.limits)
	if err != nil {
		return Result{}, err
	}
	paths, before, err := t.prepare(headers.paths)
	if err != nil {
		return Result{}, err
	}
	if err := t.checkBaselines(args.Baselines); err != nil {
		return Result{}, err
	}
	patch := []byte(args.Patch)
	if _, err := t.git.run(ctx, t.root, []string{"apply", "--check", "--whitespace=nowarn", "-"}, patch); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrPatchConflict, err)
	}
	if err := t.checkBaselines(args.Baselines); err != nil {
		return Result{}, err
	}
	if _, err := t.git.run(ctx, t.root, []string{"apply", "--whitespace=nowarn", "-"}, patch); err != nil {
		if !unchanged(before) {
			return Result{}, fmt.Errorf("%w: %v", ErrPatchAtomicityBreach, err)
		}
		return Result{}, fmt.Errorf("%w: %v", ErrPatchConflict, err)
	}
	diff, err := t.git.run(ctx, t.root, []string{"diff", "--binary"}, nil)
	if err != nil {
		return Result{}, t.recoverAfterDiffFailure(ctx, patch, before, err)
	}
	digest := sha256.Sum256(diff)
	data, _ := json.Marshal(paths)
	return Result{Code: "PATCH", Data: data, SHA256: hex.EncodeToString(digest[:])}, nil
}

func (t patchTool) recoverAfterDiffFailure(ctx context.Context, patch []byte, before map[string][32]byte, diffErr error) error {
	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := t.git.run(rollbackCtx, t.root, []string{"apply", "-R", "--whitespace=nowarn", "-"}, patch); err != nil {
		return fmt.Errorf("%w: git diff --binary: %v; reverse apply: %v", ErrPatchAtomicityBreach, diffErr, err)
	}
	if !unchanged(before) {
		return fmt.Errorf("%w: git diff --binary: %v; mutation remains after reverse apply", ErrPatchAtomicityBreach, diffErr)
	}
	return fmt.Errorf("%w: git diff --binary: %v", ErrPatchConflict, diffErr)
}

func (t patchTool) prepare(paths []string) ([]string, map[string][32]byte, error) {
	prepared := make([]string, 0, len(paths))
	before := make(map[string][32]byte, len(paths))
	for _, path := range paths {
		resolved, err := resolveToolPath(t.root, path)
		if err != nil {
			return nil, nil, err
		}
		content, err := os.ReadFile(resolved)
		if err != nil {
			return nil, nil, err
		}
		if isBinary(content) {
			return nil, nil, ErrBinaryFile
		}
		before[resolved] = sha256.Sum256(content)
		prepared = append(prepared, path)
	}
	sort.Strings(prepared)
	return prepared, before, nil
}

func (t patchTool) checkBaselines(baselines map[string]string) error {
	for path, expected := range baselines {
		resolved, err := resolveToolPath(t.root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(resolved)
		if err != nil {
			return err
		}
		actual := sha256.Sum256(content)
		if hex.EncodeToString(actual[:]) != expected {
			return fmt.Errorf("%w: %s", ErrStaleBaseline, path)
		}
	}
	return nil
}

func resolveToolPath(root, path string) (string, error) {
	resolved, err := workspace.Resolve(root, path)
	if err != nil {
		return "", normalizePathError(err)
	}
	return resolved, nil
}

func unchanged(before map[string][32]byte) bool {
	for path, expected := range before {
		content, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(content) != expected {
			return false
		}
	}
	return true
}

func boundedPatchLimits(limits PatchLimits) PatchLimits {
	if limits.MaxFiles <= 0 || limits.MaxFiles > maximumPatchFiles {
		limits.MaxFiles = maximumPatchFiles
	}
	if limits.MaxChangedLines <= 0 || limits.MaxChangedLines > maximumPatchLines {
		limits.MaxChangedLines = maximumPatchLines
	}
	return limits
}
