package tools

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/workspace"
)

type listTool struct {
	root  string
	limit int
}

func NewListTool(root string, limit int) Tool { return listTool{root: root, limit: limit} }
func (listTool) Kind() string                 { return "list_files" }

func (t listTool) Execute(ctx context.Context, raw json.RawMessage) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if !validEmptyObject(raw) {
		return Result{}, ErrInvalidArgs
	}
	paths, truncated, err := walkFiles(ctx, t.root, t.limit)
	if err != nil {
		return Result{}, err
	}
	data, _ := json.Marshal(paths)
	return Result{Code: "LIST", Data: data, Truncated: truncated}, nil
}

type SearchMatch struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}
type searchTool struct {
	root  string
	limit int
}

func NewSearchTool(root string, limit int) Tool { return searchTool{root: root, limit: limit} }
func (searchTool) Kind() string                 { return "search_text" }

func (t searchTool) Execute(ctx context.Context, raw json.RawMessage) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.Query == "" {
		return Result{}, ErrInvalidArgs
	}
	re, err := regexp.Compile(args.Query)
	if err != nil {
		return Result{}, ErrInvalidRegex
	}
	paths, _, err := walkFiles(ctx, t.root, -1)
	if err != nil {
		return Result{}, err
	}
	matches := make([]SearchMatch, 0)
	truncated := false
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		content, err := os.ReadFile(filepath.Join(t.root, filepath.FromSlash(path)))
		if err != nil || isBinary(content) {
			continue
		}
		for index, line := range strings.Split(string(content), "\n") {
			if !re.MatchString(line) {
				continue
			}
			if t.limit >= 0 && len(matches) >= t.limit {
				truncated = true
				break
			}
			matches = append(matches, SearchMatch{Path: path, Line: index + 1, Text: line})
		}
		if truncated {
			break
		}
	}
	data, _ := json.Marshal(matches)
	return Result{Code: "SEARCH", Data: data, Truncated: truncated}, nil
}

func walkFiles(ctx context.Context, root string, limit int) ([]string, bool, error) {
	root, err := workspace.ResolveRoot(root)
	if err != nil {
		return nil, false, err
	}
	paths := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() && (rel == ".git" || strings.HasPrefix(rel, ".git/")) {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if _, err := workspace.Resolve(root, rel); err != nil {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	sort.Strings(paths)
	if limit >= 0 && len(paths) > limit {
		return paths[:limit], true, nil
	}
	return paths, false, nil
}

func validEmptyObject(raw json.RawMessage) bool {
	var value map[string]json.RawMessage
	return json.Unmarshal(raw, &value) == nil && len(value) == 0
}
