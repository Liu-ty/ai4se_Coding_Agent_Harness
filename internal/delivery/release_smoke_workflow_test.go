package delivery_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type smokeWorkflow struct {
	On struct {
		WorkflowDispatch struct {
			Inputs map[string]struct {
				Required bool `yaml:"required"`
			} `yaml:"inputs"`
		} `yaml:"workflow_dispatch"`
	} `yaml:"on"`
	Jobs map[string]struct {
		RunsOn string `yaml:"runs-on"`
		Steps  []struct {
			Name string `yaml:"name"`
			Run  string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func TestReleaseSmokeWorkflowRunsLinuxBinaryOnUbuntu(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	content, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release-smoke.yml"))
	if err != nil {
		t.Fatalf("read release-smoke.yml: %v", err)
	}

	var workflow smokeWorkflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("parse release-smoke.yml: %v", err)
	}
	if !workflow.On.WorkflowDispatch.Inputs["tag"].Required {
		t.Fatal("workflow_dispatch tag input must be required")
	}
	job, exists := workflow.Jobs["verify-release"]
	if !exists {
		t.Fatal("missing verify-release job")
	}
	if job.RunsOn != "ubuntu-latest" {
		t.Fatalf("verify-release runs-on = %q, want ubuntu-latest", job.RunsOn)
	}

	var script string
	for _, step := range job.Steps {
		if step.Name == "Verify Linux release assets" {
			script = step.Run
			break
		}
	}
	if script == "" {
		t.Fatal("missing Verify Linux release assets step")
	}
	entries := strings.Split(script, "\n")
	if hasCommandEntry(entries, "cd dist", false) {
		t.Error("verification script must stay at the workspace root because checksums contain dist/ paths")
	}
	for _, command := range []string{
		`gh release download "$RELEASE_TAG" --repo "$GITHUB_REPOSITORY" --dir dist`,
		"sha256sum --check dist/checksums.txt",
		"chmod +x dist/ai4se-harness_linux_amd64",
		`dist/ai4se-harness_linux_amd64 demo feedback-loop --format json > smoke.json`,
		`grep -q '"state":"SUCCEEDED"' smoke.json`,
	} {
		if !hasCommandEntry(entries, command, false) {
			t.Errorf("verification script missing command entry %q", command)
		}
	}
}
