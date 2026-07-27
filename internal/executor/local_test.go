package executor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

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

type stubProcessTree struct {
	cleanupErr error
}

func stubProcessTreeFactory(cleanupErr error) func(*exec.Cmd) (processTreeController, error) {
	return func(*exec.Cmd) (processTreeController, error) {
		return &stubProcessTree{cleanupErr: cleanupErr}, nil
	}
}

func (*stubProcessTree) afterStart(*exec.Cmd) error { return nil }
func (s *stubProcessTree) terminate() error         { return s.cleanupErr }
func (s *stubProcessTree) close() error             { return s.cleanupErr }

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

func TestExecutorLocalHelperProcess(t *testing.T) {
	if os.Getenv("GO_EXECUTOR_LOCAL_HELPER") != "1" {
		return
	}
	os.Exit(0)
}
