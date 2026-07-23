package config

import (
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

// Config is the top-level harness configuration.
type Config struct {
	Version        int                      `toml:"version"`
	DefaultProfile domain.PermissionProfile `toml:"default_profile"`
	Budget         BudgetConfig             `toml:"budget"`
	Validation     []ValidationStage        `toml:"validation"`
	Policy         PolicyConfig             `toml:"policy"`
}

// BudgetConfig holds budget limits.
type BudgetConfig struct {
	MaxDecisions       int    `toml:"max_decisions"`
	MaxMutations       int    `toml:"max_mutations"`
	MaxProtocolRepairs int    `toml:"max_protocol_repairs"`
	WallClock          string `toml:"wall_clock"`
}

// CommandOverride holds platform-specific command overrides.
type CommandOverride struct {
	Executable string   `toml:"executable"`
	Args       []string `toml:"args"`
}

// ValidationStage is a single validation stage definition.
type ValidationStage struct {
	ID               string           `toml:"id"`
	Kind             string           `toml:"kind"`
	Executable       string           `toml:"executable"`
	Args             []string         `toml:"args"`
	WorkingDirectory string           `toml:"working_directory"`
	Timeout          string           `toml:"timeout"`
	MaxOutputBytes   int              `toml:"max_output_bytes"`
	Required         bool             `toml:"required"`
	Windows          *CommandOverride `toml:"windows"`
	Linux            *CommandOverride `toml:"linux"`
	Classifiers      []ClassifierRule `toml:"classifiers"`
}

// ClassifierRule is a regex-based failure classifier.
type ClassifierRule struct {
	Category string `toml:"category"`
	Pattern  string `toml:"pattern"`
}

// PolicyConfig holds policy-level limits.
type PolicyConfig struct {
	MaxFiles        int      `toml:"max_files"`
	MaxChangedLines int      `toml:"max_changed_lines"`
	MaxFileBytes    int      `toml:"max_file_bytes"`
	Protected       []string `toml:"protected"`
}

// CommandSpec is a resolved, platform-specific command specification.
type CommandSpec struct {
	ID               string
	Kind             string
	Executable       string
	Args             []string
	WorkingDirectory string
	Timeout          time.Duration
	MaxOutputBytes   int
	Required         bool
	Classifiers      []ClassifierRule
}
