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
	if limit >= 0 {
		return boundedFiles(ctx, root, limit)
	}
	return walkAllFiles(ctx, root)
}

func walkAllFiles(ctx context.Context, root string) ([]string, bool, error) {
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
		if entry.IsDir() && isGitDirectory(rel) {
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
	return paths, false, nil
}

type pendingPath struct {
	rel   string
	isDir bool
}

// boundedFiles expands only the lexically earliest directory prefix needed to
// identify the requested sorted file prefix plus one additional result.
func boundedFiles(ctx context.Context, root string, limit int) ([]string, bool, error) {
	root, err := workspace.ResolveRoot(root)
	if err != nil {
		return nil, false, err
	}
	pending, err := readDirectory(root, "")
	if err != nil {
		return nil, false, err
	}
	paths := make([]string, 0, limit+1)
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		sort.Slice(pending, func(i, j int) bool { return pending[i].rel < pending[j].rel })
		next := pending[0]
		pending = pending[1:]
		if next.isDir {
			if isGitDirectory(next.rel) {
				continue
			}
			if _, err := workspace.Resolve(root, strings.TrimSuffix(next.rel, "/")); err != nil {
				continue
			}
			children, err := readDirectory(root, strings.TrimSuffix(next.rel, "/"))
			if err != nil {
				return nil, false, err
			}
			pending = append(pending, children...)
			continue
		}
		if _, err := workspace.Resolve(root, next.rel); err != nil {
			continue
		}
		paths = append(paths, next.rel)
		if len(paths) > limit {
			return paths[:limit], true, nil
		}
	}
	return paths, false, nil
}

func isGitDirectory(rel string) bool {
	clean := strings.TrimSuffix(filepath.ToSlash(rel), "/")
	for _, part := range strings.Split(clean, "/") {
		if part == ".git" {
			return true
		}
	}
	return false
}

func readDirectory(root, relative string) ([]pendingPath, error) {
	directory := root
	if relative != "" {
		directory = filepath.Join(root, filepath.FromSlash(relative))
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	pending := make([]pendingPath, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		rel := entry.Name()
		if relative != "" {
			rel = relative + "/" + rel
		}
		if entry.IsDir() {
			pending = append(pending, pendingPath{rel: rel + "/", isDir: true})
		} else {
			pending = append(pending, pendingPath{rel: rel})
		}
	}
	return pending, nil
}

func validEmptyObject(raw json.RawMessage) bool {
	var value map[string]json.RawMessage
	return json.Unmarshal(raw, &value) == nil && len(value) == 0
}
