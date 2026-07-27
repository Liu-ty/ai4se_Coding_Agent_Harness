package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
)

type createTool struct {
	root  string
	limit int
}

func NewCreateTool(root string, limit int) Tool { return createTool{root: root, limit: limit} }
func (createTool) Kind() string                 { return "create_file" }

func (t createTool) Execute(ctx context.Context, raw json.RawMessage) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.Path == "" {
		return Result{}, ErrInvalidArgs
	}
	if t.limit >= 0 && len(args.Content) > t.limit {
		return Result{}, ErrCreateLimit
	}
	path, err := resolveToolPath(t.root, args.Path)
	if err != nil {
		return Result{}, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return Result{}, ErrAlreadyExists
		}
		return Result{}, err
	}
	created := true
	defer func() {
		if created {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.WriteString(args.Content); err != nil {
		_ = file.Close()
		return Result{}, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return Result{}, err
	}
	if err := file.Close(); err != nil {
		return Result{}, err
	}
	created = false
	hash := sha256.Sum256([]byte(args.Content))
	data, _ := json.Marshal([]string{args.Path})
	return Result{Code: "CREATE", Data: data, SHA256: hex.EncodeToString(hash[:])}, nil
}
