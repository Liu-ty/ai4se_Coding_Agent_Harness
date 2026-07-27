//go:build windows

package executor

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

func TestPrepareProcessTreeStartsWindowsProcessSuspended(t *testing.T) {
	cmd := exec.Command("test-helper")
	controller, err := prepareProcessTree(cmd)
	if err != nil {
		t.Fatalf("prepare process tree: %v", err)
	}
	defer controller.close()

	if cmd.SysProcAttr == nil || cmd.SysProcAttr.CreationFlags&windows.CREATE_SUSPENDED == 0 {
		t.Fatalf("creation flags = %#v, want CREATE_SUSPENDED", cmd.SysProcAttr)
	}
}
