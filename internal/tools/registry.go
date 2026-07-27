package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var (
	ErrUnknownTool          = errors.New("unknown tool")
	ErrDuplicateTool        = errors.New("duplicate tool")
	ErrInvalidArgs          = errors.New("invalid tool arguments")
	ErrBinaryFile           = errors.New("binary file")
	ErrInvalidRegex         = errors.New("invalid regular expression")
	ErrProtectedPath        = errors.New("protected path")
	ErrInvalidPatch         = errors.New("invalid patch")
	ErrPatchLimit           = errors.New("patch limit exceeded")
	ErrPatchConflict        = errors.New("patch does not apply")
	ErrStaleBaseline        = errors.New("stale baseline")
	ErrPatchAtomicityBreach = errors.New("PATCH_ATOMICITY_BREACH")
	ErrAlreadyExists        = errors.New("target already exists")
	ErrCreateLimit          = errors.New("create file limit exceeded")
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry(list ...Tool) (*Registry, error) {
	registry := &Registry{tools: make(map[string]Tool, len(list))}
	for _, tool := range list {
		if tool == nil || tool.Kind() == "" {
			return nil, fmt.Errorf("%w: empty tool", ErrDuplicateTool)
		}
		if _, exists := registry.tools[tool.Kind()]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateTool, tool.Kind())
		}
		registry.tools[tool.Kind()] = tool
	}
	return registry, nil
}

func (r *Registry) Execute(ctx context.Context, action domain.Action) (Result, error) {
	tool, ok := r.tools[action.Kind]
	if !ok {
		return Result{}, fmt.Errorf("%w: %s", ErrUnknownTool, action.Kind)
	}
	return tool.Execute(ctx, action.Args)
}
