//go:build !linux && !windows

package executor

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
)

func TestPrepareProcessTreeRejectsUnsupportedPlatform(t *testing.T) {
	cmd := exec.Command("test-helper")
	tree, err := prepareProcessTree(cmd)
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("prepare process tree error = %v, want %v", err, ErrUnsupportedPlatform)
	}
	if tree != nil {
		t.Fatalf("process tree = %#v, want nil", tree)
	}
	if cmd.SysProcAttr != nil {
		t.Fatalf("unsupported platform mutated command: %#v", cmd.SysProcAttr)
	}
}

func TestLocalFailsBeforeStartingProcessOnUnsupportedPlatform(t *testing.T) {
	got, err := NewLocal().Run(context.Background(), config.CommandSpec{
		Executable: "definitely-missing-ai4se-executable",
	})
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("run error = %v, want %v", err, ErrUnsupportedPlatform)
	}
	if got.Code != CodeStartError {
		t.Fatalf("observation code = %q, want %q", got.Code, CodeStartError)
	}
}
