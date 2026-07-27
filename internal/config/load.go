package config

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Error sentinels returned by Load.
var (
	ErrUnknownField             = errors.New("unknown configuration field")
	ErrInvalidVersion           = errors.New("version must be 1")
	ErrDuplicateStageID         = errors.New("duplicate validation stage ID")
	ErrAbsoluteWorkingDirectory = errors.New("working directory must be relative")
	ErrEmptyExecutable          = errors.New("executable must not be empty")
	ErrInvalidTimeout           = errors.New("timeout must be a positive duration string")
	ErrInvalidProfile           = errors.New("default_profile must be review, supervised, or workspace-auto")
	ErrEmptyWorkingDirectory    = errors.New("working directory must not be empty")
	ErrInvalidBudget            = errors.New("invalid budget configuration")
)

// validProfiles is the set of recognized permission profile values.
var validProfiles = map[string]bool{
	"review":         true,
	"supervised":     true,
	"workspace-auto": true,
}

// Load reads and validates a TOML configuration from r.
// It rejects unknown fields, incorrect version, duplicate stage IDs,
// absolute working directories, empty executables, and invalid timeouts.
func Load(r io.Reader) (Config, error) {
	cfg := Config{
		Budget: BudgetConfig{
			MaxDecisions:       30,
			MaxMutations:       5,
			MaxProtocolRepairs: 2,
			WallClock:          "20m",
		},
	}
	md, err := toml.NewDecoder(r).Decode(&cfg)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	// Reject unknown keys.
	if len(md.Undecoded()) > 0 {
		keys := make([]string, len(md.Undecoded()))
		for i, k := range md.Undecoded() {
			keys[i] = k.String()
		}
		return Config{}, fmt.Errorf("%w: %s", ErrUnknownField, strings.Join(keys, ", "))
	}

	// Validate version.
	if cfg.Version != 1 {
		return Config{}, ErrInvalidVersion
	}

	// Validate default profile.
	if !validProfiles[string(cfg.DefaultProfile)] {
		return Config{}, fmt.Errorf("%w: %q", ErrInvalidProfile, cfg.DefaultProfile)
	}

	if err := validateBudget(cfg.Budget); err != nil {
		return Config{}, err
	}

	// Validate validation stages.
	seen := make(map[string]bool)
	for i := range cfg.Validation {
		stage := &cfg.Validation[i]

		// Duplicate IDs.
		if seen[stage.ID] {
			return Config{}, fmt.Errorf("%w: %q", ErrDuplicateStageID, stage.ID)
		}
		seen[stage.ID] = true

		// Empty working directory.
		if stage.WorkingDirectory == "" {
			return Config{}, fmt.Errorf("%w: stage %q", ErrEmptyWorkingDirectory, stage.ID)
		}

		// Reject absolute paths using both host and cross-platform syntax.
		if isAbsoluteWorkingDirectory(stage.WorkingDirectory) {
			return Config{}, fmt.Errorf("%w: %q", ErrAbsoluteWorkingDirectory, stage.WorkingDirectory)
		}

		// Empty base executable.
		if stage.Executable == "" {
			return Config{}, fmt.Errorf("%w: stage %q base executable", ErrEmptyExecutable, stage.ID)
		}

		// Empty override executables.
		if stage.Windows != nil && stage.Windows.Executable == "" {
			return Config{}, fmt.Errorf("%w: stage %q windows override executable", ErrEmptyExecutable, stage.ID)
		}
		if stage.Linux != nil && stage.Linux.Executable == "" {
			return Config{}, fmt.Errorf("%w: stage %q linux override executable", ErrEmptyExecutable, stage.ID)
		}

		// Validate timeout.
		if _, err := parsePositiveDuration(stage.Timeout); err != nil {
			return Config{}, fmt.Errorf("%w: stage %q: %w", ErrInvalidTimeout, stage.ID, err)
		}
	}

	return cfg, nil
}

func validateBudget(budget BudgetConfig) error {
	switch {
	case budget.MaxDecisions <= 0:
		return fmt.Errorf("%w: max_decisions must be positive", ErrInvalidBudget)
	case budget.MaxMutations <= 0:
		return fmt.Errorf("%w: max_mutations must be positive", ErrInvalidBudget)
	case budget.MaxProtocolRepairs <= 0:
		return fmt.Errorf("%w: max_protocol_repairs must be positive", ErrInvalidBudget)
	}
	wallClock, err := time.ParseDuration(budget.WallClock)
	if err != nil {
		return fmt.Errorf("%w: wall_clock: %v", ErrInvalidBudget, err)
	}
	if wallClock <= 0 {
		return fmt.Errorf("%w: wall_clock must be positive", ErrInvalidBudget)
	}
	return nil
}

func isAbsoluteWorkingDirectory(path string) bool {
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) {
		return true
	}

	return len(path) >= 2 &&
		((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) &&
		path[1] == ':'
}

// parsePositiveDuration parses a duration string and returns an error
// if the value is zero, negative, or malformed.
func parsePositiveDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, fmt.Errorf("timeout must be positive, got %s", s)
	}
	return d, nil
}
