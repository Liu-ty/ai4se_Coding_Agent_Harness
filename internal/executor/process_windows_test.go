//go:build windows

package executor

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

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

func TestWindowsProcessTreeHandlesImmediateTermination(t *testing.T) {
	t.Setenv("GO_EXECUTOR_WINDOWS_HELPER", "1")
	for i := 0; i < 25; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestWindowsProcessTreeChild")
		controller, err := prepareProcessTree(cmd)
		if err != nil {
			cancel()
			t.Fatalf("prepare process tree: %v", err)
		}
		if err := cmd.Start(); err != nil {
			_ = controller.close()
			cancel()
			t.Fatalf("start suspended child: %v", err)
		}

		afterStart := make(chan error, 1)
		terminated := make(chan error, 1)
		go func() { afterStart <- controller.afterStart(cmd) }()
		go func() { terminated <- controller.terminate() }()
		if err := <-afterStart; err != nil {
			t.Fatalf("after start: %v", err)
		}
		if err := <-terminated; err != nil {
			t.Fatalf("terminate: %v", err)
		}
		_ = cmd.Wait()
		if err := controller.close(); err != nil {
			t.Fatalf("close: %v", err)
		}
		cancel()
	}
}

func TestWindowsProcessTreeChild(t *testing.T) {
	if os.Getenv("GO_EXECUTOR_WINDOWS_HELPER") != "1" {
		return
	}
	os.Exit(0)
}
