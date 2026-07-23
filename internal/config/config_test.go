package config_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
)

var _ func(io.Reader) (config.Config, error) = config.Load

func TestLoadRejectsUnknownField(t *testing.T) {
	_, err := config.Load(strings.NewReader("version = 1\nunknown = true\n"))
	if !errors.Is(err, config.ErrUnknownField) {
		t.Fatalf("expected ErrUnknownField, got %v", err)
	}
}

func TestLoadRejectsWrongVersion(t *testing.T) {
	_, err := config.Load(strings.NewReader("version = 2\n"))
	if !errors.Is(err, config.ErrInvalidVersion) {
		t.Fatalf("expected ErrInvalidVersion, got %v", err)
	}
}

func TestLoadRejectsInvalidProfile(t *testing.T) {
	_, err := config.Load(strings.NewReader("version = 1\ndefault_profile = \"garbage\"\n"))
	if !errors.Is(err, config.ErrInvalidProfile) {
		t.Fatalf("expected ErrInvalidProfile, got %v", err)
	}
}

func TestLoadRejectsDuplicateStageIDs(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "dup"
executable = "go"
working_directory = "."
timeout = "1m"
[[validation]]
id = "dup"
executable = "go"
working_directory = "."
timeout = "1m"
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrDuplicateStageID) {
		t.Fatalf("expected ErrDuplicateStageID, got %v", err)
	}
}

func TestLoadRejectsEmptyWorkingDirectory(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "test"
executable = "go"
working_directory = ""
timeout = "1m"
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrEmptyWorkingDirectory) {
		t.Fatalf("expected ErrEmptyWorkingDirectory, got %v", err)
	}
}

func TestLoadRejectsAbsolutePOSIXWorkingDirectory(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "test"
executable = "go"
working_directory = "/absolute/path"
timeout = "1m"
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrAbsoluteWorkingDirectory) {
		t.Fatalf("expected ErrAbsoluteWorkingDirectory, got %v", err)
	}
}

func TestLoadRejectsCrossPlatformAbsoluteWorkingDirectories(t *testing.T) {
	paths := []string{
		`C:\absolute\path`,
		`C:/absolute/path`,
		`C:relative`,
		`c:relative`,
		`\\server\share`,
		`\rooted`,
		`/absolute/path`,
	}

	for _, workingDirectory := range paths {
		t.Run(workingDirectory, func(t *testing.T) {
			input := "version = 1\n" +
				"default_profile = \"workspace-auto\"\n" +
				"[[validation]]\n" +
				"id = \"test\"\n" +
				"executable = \"go\"\n" +
				"working_directory = " + strconv.Quote(workingDirectory) + "\n" +
				"timeout = \"1m\"\n"

			_, err := config.Load(strings.NewReader(input))
			if !errors.Is(err, config.ErrAbsoluteWorkingDirectory) {
				t.Fatalf("expected ErrAbsoluteWorkingDirectory for %q, got %v", workingDirectory, err)
			}
		})
	}
}

func TestLoadRejectsEmptyExecutable(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "test"
executable = ""
working_directory = "."
timeout = "1m"
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrEmptyExecutable) {
		t.Fatalf("expected ErrEmptyExecutable, got %v", err)
	}
}

func TestLoadRejectsEmptyOverrideExecutable(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "test"
executable = "go"
working_directory = "."
timeout = "1m"

[validation.windows]
executable = ""
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrEmptyExecutable) {
		t.Fatalf("expected ErrEmptyExecutable, got %v", err)
	}
}

func TestLoadRejectsMalformedTimeout(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "test"
executable = "go"
working_directory = "."
timeout = "not-a-duration"
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrInvalidTimeout) {
		t.Fatalf("expected ErrInvalidTimeout, got %v", err)
	}
}

func TestLoadRejectsZeroTimeout(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "test"
executable = "go"
working_directory = "."
timeout = "0s"
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrInvalidTimeout) {
		t.Fatalf("expected ErrInvalidTimeout, got %v", err)
	}
}

func TestLoadRejectsNegativeTimeout(t *testing.T) {
	toml := `
version = 1
default_profile = "workspace-auto"
[[validation]]
id = "test"
executable = "go"
working_directory = "."
timeout = "-1m"
`
	_, err := config.Load(strings.NewReader(toml))
	if !errors.Is(err, config.ErrInvalidTimeout) {
		t.Fatalf("expected ErrInvalidTimeout, got %v", err)
	}
}

func TestResolveWindowsOverride(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Args:       []string{"test", "./..."},
		Timeout:    "1m",
		Windows:    &config.CommandOverride{Executable: "go.exe"},
	}
	got, err := config.ResolveStage(stage, "windows")
	if err != nil || got.Executable != "go.exe" {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestResolveLinuxOverride(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Args:       []string{"test", "./..."},
		Timeout:    "1m",
		Linux:      &config.CommandOverride{Executable: "my-go"},
	}
	got, err := config.ResolveStage(stage, "linux")
	if err != nil || got.Executable != "my-go" {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestResolveUsesBaseWhenNoOverride(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Args:       []string{"test", "./..."},
		Timeout:    "1m",
		Windows:    &config.CommandOverride{Executable: "go.exe"},
	}
	got, err := config.ResolveStage(stage, "linux")
	if err != nil || got.Executable != "go" {
		t.Fatalf("expected base executable 'go' for linux, got %#v, %v", got, err)
	}
}

func TestResolvePreservesArgsWhenNoOverride(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Args:       []string{"test", "./..."},
		Timeout:    "1m",
	}
	got, err := config.ResolveStage(stage, "linux")
	if err != nil || len(got.Args) != 2 || got.Args[0] != "test" || got.Args[1] != "./..." {
		t.Fatalf("expected args preserved, got %#v, %v", got, err)
	}
}

func TestResolveOverrideArgsReplaceBase(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Args:       []string{"test", "./..."},
		Timeout:    "1m",
		Windows:    &config.CommandOverride{Executable: "go.exe", Args: []string{"test", "-v", "./..."}},
	}
	got, err := config.ResolveStage(stage, "windows")
	if err != nil || len(got.Args) != 3 || got.Args[0] != "test" || got.Args[1] != "-v" {
		t.Fatalf("expected overridden args, got %#v, %v", got, err)
	}
}

func TestLoadValidCanonicalFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "config", "valid.toml")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open fixture: %v", err)
	}
	defer f.Close()
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected version 1, got %d", cfg.Version)
	}
	if cfg.DefaultProfile != "workspace-auto" {
		t.Fatalf("expected workspace-auto profile, got %s", cfg.DefaultProfile)
	}
	if cfg.Budget.MaxDecisions != 30 {
		t.Fatalf("expected 30 max decisions, got %d", cfg.Budget.MaxDecisions)
	}
	if cfg.Budget.MaxMutations != 5 {
		t.Fatalf("expected 5 max mutations, got %d", cfg.Budget.MaxMutations)
	}
	if cfg.Budget.MaxProtocolRepairs != 2 {
		t.Fatalf("expected 2 max protocol repairs, got %d", cfg.Budget.MaxProtocolRepairs)
	}
	if cfg.Budget.WallClock != "20m" {
		t.Fatalf("expected 20m wall clock, got %s", cfg.Budget.WallClock)
	}
	if len(cfg.Validation) != 1 {
		t.Fatalf("expected 1 validation stage, got %d", len(cfg.Validation))
	}
	stage := cfg.Validation[0]
	if stage.ID != "unit-test" {
		t.Fatalf("expected unit-test id, got %s", stage.ID)
	}
	if stage.Kind != "targeted-test" {
		t.Fatalf("expected targeted-test kind, got %s", stage.Kind)
	}
	if stage.Executable != "go" {
		t.Fatalf("expected go executable, got %s", stage.Executable)
	}
	if stage.WorkingDirectory != "." {
		t.Fatalf("expected . working directory, got %s", stage.WorkingDirectory)
	}
	if stage.Timeout != "2m" {
		t.Fatalf("expected 2m timeout, got %s", stage.Timeout)
	}
	if stage.MaxOutputBytes != 262144 {
		t.Fatalf("expected 262144 max output bytes, got %d", stage.MaxOutputBytes)
	}
	if !stage.Required {
		t.Fatal("expected required to be true")
	}
	if stage.Windows == nil || stage.Windows.Executable != "go.exe" {
		t.Fatalf("expected windows override with go.exe, got %#v", stage.Windows)
	}
}

func TestResolveStagePropagatesTimeout(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Args:       []string{"test", "./..."},
		Timeout:    "2m",
	}
	got, err := config.ResolveStage(stage, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Timeout.Seconds() != 120 {
		t.Fatalf("expected 120s timeout, got %v", got.Timeout)
	}
}

func TestResolveStageRejectsInvalidTimeout(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Timeout:    "0s",
	}
	_, err := config.ResolveStage(stage, "linux")
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestResolveStagePreservesClassifiers(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Timeout:    "1m",
		Classifiers: []config.ClassifierRule{
			{Category: "test", Pattern: "FAIL"},
		},
	}
	got, err := config.ResolveStage(stage, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Classifiers) != 1 || got.Classifiers[0].Category != "test" {
		t.Fatalf("expected classifiers preserved, got %#v", got.Classifiers)
	}
}

func TestResolveDarwinUsesBase(t *testing.T) {
	stage := config.ValidationStage{
		ID:         "unit-test",
		Executable: "go",
		Args:       []string{"test", "./..."},
		Timeout:    "1m",
		Windows:    &config.CommandOverride{Executable: "go.exe"},
		Linux:      &config.CommandOverride{Executable: "my-go"},
	}
	got, err := config.ResolveStage(stage, "darwin")
	if err != nil || got.Executable != "go" {
		t.Fatalf("expected base executable 'go' for darwin, got %#v, %v", got, err)
	}
}
