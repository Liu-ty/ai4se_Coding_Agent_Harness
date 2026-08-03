package delivery_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type job struct {
	Image        string   `yaml:"image"`
	BeforeScript []string `yaml:"before_script"`
	Script       []string `yaml:"script"`
	Cache        struct {
		Key   string   `yaml:"key"`
		Paths []string `yaml:"paths"`
	} `yaml:"cache"`
}

type requiredCommand struct {
	Text                string
	AllowAdditionalArgs bool
}

func hasCommandEntry(entries []string, required string, allowAdditionalArgs bool) bool {
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == required || (allowAdditionalArgs && strings.HasPrefix(entry, required+" ")) {
			return true
		}
	}
	return false
}

func TestHasCommandEntryRejectsEchoedCommand(t *testing.T) {
	entries := []string{`echo "go test ./... -count=1"`}

	if hasCommandEntry(entries, "go test ./... -count=1", false) {
		t.Fatal("echoed text must not satisfy a required command entry")
	}
}

func TestGitLabCIContainsRequiredJobs(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	content, err := os.ReadFile(filepath.Join(root, ".gitlab-ci.yml"))
	if err != nil {
		t.Fatalf("read .gitlab-ci.yml: %v", err)
	}

	var config map[string]yaml.Node
	if err := yaml.Unmarshal(content, &config); err != nil {
		t.Fatalf("parse .gitlab-ci.yml: %v", err)
	}

	assertJob := func(name, image, cacheKey, cachePath string, commands ...requiredCommand) {
		t.Helper()
		node, exists := config[name]
		if !exists {
			t.Fatalf("missing %s job", name)
		}
		var got job
		if err := node.Decode(&got); err != nil {
			t.Fatalf("decode %s job: %v", name, err)
		}
		if got.Image != image {
			t.Fatalf("%s image = %q, want %q", name, got.Image, image)
		}
		if got.Cache.Key != cacheKey {
			t.Errorf("%s cache key = %q, want %q", name, got.Cache.Key, cacheKey)
		}
		if len(got.Cache.Paths) != 1 || got.Cache.Paths[0] != cachePath {
			t.Errorf("%s cache paths = %q, want [%q]", name, got.Cache.Paths, cachePath)
		}
		commandsInJob := append(append([]string(nil), got.BeforeScript...), got.Script...)
		for _, command := range commands {
			if !hasCommandEntry(commandsInJob, command.Text, command.AllowAdditionalArgs) {
				t.Errorf("%s script missing command entry %q", name, command.Text)
			}
		}
	}

	assertJob("unit-test", "golang:1.26.5-bookworm", "go-$CI_COMMIT_REF_SLUG", ".cache/go/",
		requiredCommand{Text: "go test ./... -count=1"},
		requiredCommand{Text: "go vet ./..."},
	)
	assertJob("frontend-test", "node:24-bookworm", "npm-$CI_COMMIT_REF_SLUG", ".cache/npm/",
		requiredCommand{Text: "npm --prefix web ci", AllowAdditionalArgs: true},
		requiredCommand{Text: "npm --prefix web test -- --run"},
		requiredCommand{Text: "npm --prefix web run build"},
	)
}
