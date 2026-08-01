package demo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCIWorkflowHasCrossPlatformUnitTestJob(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Strategy struct {
				Matrix struct {
					OS []string `yaml:"os"`
				} `yaml:"matrix"`
			} `yaml:"strategy"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(contents, &workflow); err != nil {
		t.Fatal(err)
	}
	unit, ok := workflow.Jobs["unit-test"]
	if !ok {
		t.Fatal("CI must define a job key exactly named unit-test")
	}
	want := map[string]bool{"ubuntu-latest": true, "windows-latest": true}
	for _, value := range unit.Strategy.Matrix.OS {
		delete(want, value)
	}
	if len(want) != 0 {
		t.Fatalf("unit-test matrix missing %v", want)
	}
}

func TestCIWorkflowPreparesPinnedBrowserAndGitleaksToken(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Uses string            `yaml:"uses"`
				Run  string            `yaml:"run"`
				Env  map[string]string `yaml:"env"`
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(contents, &workflow); err != nil {
		t.Fatal(err)
	}
	if !hasRun(workflow.Jobs["e2e"].Steps, "npx --prefix web playwright install --with-deps chromium") {
		t.Fatal("e2e must install Chromium through web's lockfile-pinned Playwright")
	}
	steps := workflow.Jobs["security-test"].Steps
	for _, step := range steps {
		if strings.HasPrefix(step.Uses, "gitleaks/gitleaks-action@") && step.Env["GITHUB_TOKEN"] != "${{ secrets.GITHUB_TOKEN }}" {
			t.Fatal("gitleaks pull-request scan must receive GITHUB_TOKEN")
		}
	}
	if !hasCheckoutHistory(steps) {
		t.Fatal("gitleaks pull-request scan must fetch the PR baseline history")
	}
	for _, jobName := range []string{"frontend-test", "security-test", "integration-test", "build", "docker-build", "e2e"} {
		if !hasCredentialSafeCheckout(workflow.Jobs[jobName].Steps) {
			t.Fatalf("%s must disable checkout credential persistence", jobName)
		}
	}
}

func hasRun(steps []struct {
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
	With map[string]string `yaml:"with"`
}, want string) bool {
	for _, step := range steps {
		if step.Run == want {
			return true
		}
	}
	return false
}

func hasCheckoutHistory(steps []struct {
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
	With map[string]string `yaml:"with"`
}) bool {
	for _, step := range steps {
		if strings.HasPrefix(step.Uses, "actions/checkout@") && step.With["fetch-depth"] == "0" {
			return true
		}
	}
	return false
}

func hasCredentialSafeCheckout(steps []struct {
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
	With map[string]string `yaml:"with"`
}) bool {
	for _, step := range steps {
		if strings.HasPrefix(step.Uses, "actions/checkout@") && step.With["persist-credentials"] == "false" {
			return true
		}
	}
	return false
}
