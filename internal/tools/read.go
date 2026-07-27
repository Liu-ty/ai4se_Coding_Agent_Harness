package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/workspace"
)

type readTool struct {
	root  string
	limit int
}

func NewReadTool(root string, limit int) Tool { return readTool{root: root, limit: limit} }
func (readTool) Kind() string                 { return "read_file" }

func (t readTool) Execute(ctx context.Context, raw json.RawMessage) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.Path == "" {
		return Result{}, ErrInvalidArgs
	}
	path, err := workspace.Resolve(t.root, args.Path)
	if err != nil {
		return Result{}, normalizePathError(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	if isBinary(content) {
		return Result{}, ErrBinaryFile
	}
	hash := sha256.Sum256(content)
	result := Result{Code: "READ", Text: string(content), SHA256: hex.EncodeToString(hash[:])}
	if t.limit >= 0 && len(content) > t.limit {
		result.Text, result.Truncated = string(truncateUTF8(content, t.limit)), true
	}
	return result, nil
}

func truncateUTF8(content []byte, limit int) []byte {
	for limit > 0 && !utf8.RuneStart(content[limit]) {
		limit--
	}
	return content[:limit]
}

func isBinary(content []byte) bool { return !utf8.Valid(content) || containsNUL(content) }
func containsNUL(content []byte) bool {
	for _, b := range content {
		if b == 0 {
			return true
		}
	}
	return false
}
func normalizePathError(err error) error {
	if errors.Is(err, workspace.ErrProtectedPath) {
		return ErrProtectedPath
	}
	return fmt.Errorf("resolve path: %w", err)
}
