package demo

import (
	"os"
	"path/filepath"
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
