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

	assertJob := func(name, image string, commands ...string) {
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
		commandsInJob := append(append([]string(nil), got.BeforeScript...), got.Script...)
		joined := strings.Join(commandsInJob, "\n")
		for _, command := range commands {
			if !strings.Contains(joined, command) {
				t.Errorf("%s script missing %q", name, command)
			}
		}
	}

	assertJob("unit-test", "golang:1.26.5-bookworm", "go test ./... -count=1", "go vet ./...")
	assertJob("frontend-test", "node:24-bookworm", "npm --prefix web ci", "npm --prefix web test -- --run", "npm --prefix web run build")
}
