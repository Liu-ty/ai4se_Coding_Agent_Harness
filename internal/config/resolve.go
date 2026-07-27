package config

import (
	"fmt"
	"runtime"
)

// ResolveStage resolves a ValidationStage into a CommandSpec for the given
// target OS. It selects the matching override, retains base fields not
// replaced by the override, and returns the parsed positive timeout.
// It never converts an invalid timeout to zero.
// Non-windows, non-linux OSes use the base configuration.
func ResolveStage(stage ValidationStage, goos string) (CommandSpec, error) {
	executable := stage.Executable
	args := stage.Args

	if goos == "windows" && stage.Windows != nil {
		executable = stage.Windows.Executable
		if stage.Windows.Args != nil {
			args = stage.Windows.Args
		}
	} else if goos == "linux" && stage.Linux != nil {
		executable = stage.Linux.Executable
		if stage.Linux.Args != nil {
			args = stage.Linux.Args
		}
	}

	timeout, err := parsePositiveDuration(stage.Timeout)
	if err != nil {
		return CommandSpec{}, fmt.Errorf("resolve stage %q: %w", stage.ID, err)
	}

	return CommandSpec{
		ID:               stage.ID,
		Kind:             stage.Kind,
		Executable:       executable,
		Args:             args,
		WorkingDirectory: stage.WorkingDirectory,
		Timeout:          timeout,
		MaxOutputBytes:   stage.MaxOutputBytes,
		Required:         stage.Required,
		Classifiers:      stage.Classifiers,
	}, nil
}

// ResolveStageForHost resolves the stage for the current runtime OS.
func ResolveStageForHost(stage ValidationStage) (CommandSpec, error) {
	return ResolveStage(stage, runtime.GOOS)
}
