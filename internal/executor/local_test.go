package executor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
)

func TestInterpretWaitErrorReportsUnexpectedFailure(t *testing.T) {
	waitErr := errors.New("wait failed")
	exitCode, err := interpretWaitError(waitErr, false)
	if exitCode != nil {
		t.Fatalf("exit code = %d, want nil", *exitCode)
	}
	if !errors.Is(err, waitErr) {
		t.Fatalf("error = %v, want wrapped %v", err, waitErr)
	}
}

func TestInterpretWaitErrorAllowsContextInterruption(t *testing.T) {
	exitCode, err := interpretWaitError(errors.New("context canceled"), true)
	if err != nil || exitCode != nil {
		t.Fatalf("exit code/error = %v/%v, want nil/nil", exitCode, err)
	}
}

func TestLocalReportsUnexpectedWaitErrorInObservation(t *testing.T) {
	waitErr := errors.New("device wait failed")
	local := &Local{
		processTreeFactory: stubProcessTreeFactory(nil),
		waitCommand: func(cmd *exec.Cmd) error {
			_ = cmd.Wait()
			return waitErr
		},
	}

	obs, err := local.Run(context.Background(), localHelperSpec(t))

	if !errors.Is(err, waitErr) {
		t.Fatalf("error = %v, want wrapped %v", err, waitErr)
	}
	if obs.Code != CodeExecutionError || !strings.Contains(string(obs.Data), "device wait failed") {
		t.Fatalf("observation = %#v, want visible execution diagnostic", obs)
	}
}

func TestLocalReportsProcessCleanupFailure(t *testing.T) {
	cleanupErr := errors.New("cleanup syscall failed")
	local := &Local{processTreeFactory: stubProcessTreeFactory(cleanupErr)}

	obs, err := local.Run(context.Background(), localHelperSpec(t))

	if !errors.Is(err, cleanupErr) {
		t.Fatalf("error = %v, want wrapped %v", err, cleanupErr)
	}
	if obs.Code != CodeCleanupError || !strings.Contains(string(obs.Data), "cleanup syscall failed") {
		t.Fatalf("observation = %#v, want visible cleanup diagnostic", obs)
	}
}

func TestLocalBoundsDrainWhenCleanupLeavesInheritedPipesOpen(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "child.pid")
	t.Cleanup(func() { killMarkedProcess(marker) })
	local := &Local{processTreeFactory: stubProcessTreeFactory(nil)}
	spec := localInheritedPipeSpec(t, marker)
	spec.Timeout = 10 * time.Second

	started := time.Now()
	obs, err := local.Run(context.Background(), spec)

	if !errors.Is(err, ErrProcessCleanup) {
		t.Fatalf("error = %v, want %v; observation=%#v", err, ErrProcessCleanup, obs)
	}
	if obs.Code != CodeCleanupError || !obs.OutputTruncated {
		t.Fatalf("observation = %#v, want terminal truncated cleanup failure", obs)
	}
	if elapsed := time.Since(started); elapsed >= 3500*time.Millisecond {
		t.Fatalf("executor waited %v for an inherited pipe after cleanup", elapsed)
	}
}

func TestLocalAfterStartFailureDoesNotWaitForInheritedPipes(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "child.pid")
	t.Cleanup(func() { killMarkedProcess(marker) })
	afterStartErr := errors.New("assign process tree failed")
	local := &Local{
		processTreeFactory: func(*exec.Cmd) (processTreeController, error) {
			return &stubProcessTree{
				afterStartHook: func() error {
					waitForMarker(t, marker)
					return afterStartErr
				},
				cleanupErr: errors.New("cleanup after assign failed"),
			}, nil
		},
	}

	started := time.Now()
	_, err := local.Run(context.Background(), localInheritedPipeSpec(t, marker))

	if !errors.Is(err, afterStartErr) {
		t.Fatalf("error = %v, want %v", err, afterStartErr)
	}
	if elapsed := time.Since(started); elapsed >= 3500*time.Millisecond {
		t.Fatalf("after-start failure waited %v for an inherited pipe", elapsed)
	}
}

type stubProcessTree struct {
	afterStartHook func() error
	cleanupErr     error
}

func stubProcessTreeFactory(cleanupErr error) func(*exec.Cmd) (processTreeController, error) {
	return func(*exec.Cmd) (processTreeController, error) {
		return &stubProcessTree{cleanupErr: cleanupErr}, nil
	}
}

func (s *stubProcessTree) afterStart(*exec.Cmd) error {
	if s.afterStartHook != nil {
		return s.afterStartHook()
	}
	return nil
}
func (s *stubProcessTree) terminate() error { return s.cleanupErr }
func (s *stubProcessTree) close() error     { return s.cleanupErr }

func localHelperSpec(t *testing.T) config.CommandSpec {
	t.Helper()
	t.Setenv("GO_EXECUTOR_LOCAL_HELPER", "1")
	return config.CommandSpec{
		ID:             "local-helper",
		Executable:     os.Args[0],
		Args:           []string{"-test.run=TestExecutorLocalHelperProcess"},
		MaxOutputBytes: 1024,
		Required:       true,
	}
}

func localInheritedPipeSpec(t *testing.T, marker string) config.CommandSpec {
	t.Helper()
	t.Setenv("GO_EXECUTOR_LOCAL_HELPER", "spawn-inherited")
	t.Setenv("GO_EXECUTOR_LOCAL_MARKER", marker)
	return config.CommandSpec{
		ID:             "local-inherited-pipe-helper",
		Executable:     os.Args[0],
		Args:           []string{"-test.run=TestExecutorLocalHelperProcess"},
		MaxOutputBytes: 1024,
		Required:       true,
	}
}

func TestExecutorLocalHelperProcess(t *testing.T) {
	if os.Getenv("GO_EXECUTOR_LOCAL_CHILD") == "1" {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	switch os.Getenv("GO_EXECUTOR_LOCAL_HELPER") {
	case "1":
		os.Exit(0)
	case "spawn-inherited":
		child := exec.Command(os.Args[0], "-test.run=TestExecutorLocalHelperProcess")
		child.Env = append(os.Environ(), "GO_EXECUTOR_LOCAL_CHILD=1")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		marker := os.Getenv("GO_EXECUTOR_LOCAL_MARKER")
		if err := os.WriteFile(marker, []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
			_ = child.Process.Kill()
			os.Exit(3)
		}
		os.Exit(0)
	default:
		return
	}
}

func waitForMarker(t *testing.T, marker string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("child marker %s was not written", marker)
}

func killMarkedProcess(marker string) {
	data, err := os.ReadFile(marker)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = process.Kill()
	_ = process.Release()
}
