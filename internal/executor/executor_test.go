package executor_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/executor"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
)

func TestExecutorCapturesSeparateStreamsAndExitCode(t *testing.T) {
	got, err := executor.NewLocal().Run(context.Background(), helperSpec(t, "streams"))
	if err != nil {
		t.Fatalf("run streams: %v", err)
	}
	if got.Stdout != "out\n" || got.Stderr != "err\n" || got.ExitCode == nil || *got.ExitCode != 7 {
		t.Fatalf("observation = %#v", got)
	}
}

func TestExecutorDrainsBothStreamsBeforeReturning(t *testing.T) {
	spec := helperSpec(t, "stream-burst")
	spec.MaxOutputBytes = 1024 * 1024
	wantStdout := strings.Repeat("O", spec.MaxOutputBytes)
	wantStderr := strings.Repeat("E", spec.MaxOutputBytes)

	for attempt := 0; attempt < 20; attempt++ {
		got, err := executor.NewLocal().Run(context.Background(), spec)
		if err != nil {
			t.Fatalf("attempt %d: run stream burst: %v", attempt, err)
		}
		if got.Stdout != wantStdout || got.Stderr != wantStderr || got.ExitCode == nil || *got.ExitCode != 7 {
			t.Fatalf(
				"attempt %d: stdout/stderr lengths = %d/%d, exit = %v; want %d/%d and 7",
				attempt,
				len(got.Stdout),
				len(got.Stderr),
				got.ExitCode,
				len(wantStdout),
				len(wantStderr),
			)
		}
	}
}

func TestExecutorBoundsStdoutAndStderrIndependently(t *testing.T) {
	spec := helperSpec(t, "large-output")
	spec.MaxOutputBytes = 12

	got, err := executor.NewLocal().Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("run large output: %v", err)
	}
	if len(got.Stdout) != spec.MaxOutputBytes || len(got.Stderr) != spec.MaxOutputBytes {
		t.Fatalf("stdout/stderr lengths = %d/%d, want %d/%d", len(got.Stdout), len(got.Stderr), spec.MaxOutputBytes, spec.MaxOutputBytes)
	}
	if !got.OutputTruncated {
		t.Fatalf("output truncation not marked: %#v", got)
	}
	structured := feedback.Pipeline{MaxEvidence: 100, MaxSummaryBytes: 100}.Process(feedback.Input{
		StageID:     spec.ID,
		Code:        got.Code,
		Observation: got,
	})
	if !structured.OutputTruncated {
		t.Fatalf("executor truncation not preserved by feedback pipeline: %#v", structured)
	}
	if !strings.HasPrefix(got.Stdout, "OOOO") || !strings.HasPrefix(got.Stderr, "EEEE") {
		t.Fatalf("unexpected bounded streams: stdout=%q stderr=%q", got.Stdout, got.Stderr)
	}
}

func TestExecutorMarksEitherTruncatedStream(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "stdout", mode: "large-stdout"},
		{name: "stderr", mode: "large-stderr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := helperSpec(t, tt.mode)
			spec.MaxOutputBytes = 12

			got, err := executor.NewLocal().Run(context.Background(), spec)
			if err != nil {
				t.Fatalf("run %s: %v", tt.mode, err)
			}
			if !got.OutputTruncated {
				t.Fatalf("%s truncation not marked: %#v", tt.name, got)
			}
		})
	}
}

func TestExecutorDoesNotForwardSecretEnvironment(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "canary-secret")
	t.Setenv("SAFE_EXECUTOR_TEST_VALUE", "safe-value")

	got, err := executor.NewLocal().Run(context.Background(), helperSpec(t, "env"))
	if err != nil {
		t.Fatalf("run env: %v", err)
	}
	if strings.Contains(got.Stdout, "canary-secret") {
		t.Fatalf("secret inherited in stdout: %q", got.Stdout)
	}
	if !strings.Contains(got.Stdout, "SAFE_EXECUTOR_TEST_VALUE=safe-value") {
		t.Fatalf("safe environment not inherited: %q", got.Stdout)
	}
}

func TestExecutorTimeoutCleansUpProcessTree(t *testing.T) {
	dir := t.TempDir()
	heartbeat := filepath.Join(dir, "heartbeat")
	spec := helperSpec(t, "spawn-child", heartbeat)

	done := make(chan struct {
		obsCode string
		err     error
	}, 1)
	go func() {
		got, err := executor.NewLocal().Run(context.Background(), spec)
		done <- struct {
			obsCode string
			err     error
		}{obsCode: got.Code, err: err}
	}()

	waitForHeartbeat(t, heartbeat)
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("run spawn-child: %v", got.err)
		}
		if got.obsCode != executor.CodeTimeout {
			t.Fatalf("code = %q, want %q", got.obsCode, executor.CodeTimeout)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not return after timeout")
	}
	assertHeartbeatStopped(t, heartbeat)
}

func TestExecutorContextCancellationCleansUpProcessTree(t *testing.T) {
	dir := t.TempDir()
	heartbeat := filepath.Join(dir, "heartbeat")
	spec := helperSpec(t, "spawn-child", heartbeat)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct {
		obsCode string
		err     error
	}, 1)
	go func() {
		got, err := executor.NewLocal().Run(ctx, spec)
		done <- struct {
			obsCode string
			err     error
		}{obsCode: got.Code, err: err}
	}()

	waitForHeartbeat(t, heartbeat)
	cancel()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("run after cancel: %v", got.err)
		}
		if got.obsCode != executor.CodeCancelled {
			t.Fatalf("code = %q, want %q", got.obsCode, executor.CodeCancelled)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("executor did not return after cancellation")
	}
	assertHeartbeatStopped(t, heartbeat)
}

func TestExecutorExternalDeadlineIsTimeout(t *testing.T) {
	dir := t.TempDir()
	heartbeat := filepath.Join(dir, "heartbeat")
	spec := helperSpec(t, "spawn-child", heartbeat)
	spec.Timeout = 0
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := executor.NewLocal().Run(ctx, spec)
	if err != nil {
		t.Fatalf("run after external deadline: %v", err)
	}
	if got.Code != executor.CodeTimeout {
		t.Fatalf("code = %q, want %q; observation=%#v", got.Code, executor.CodeTimeout, got)
	}
	assertHeartbeatStopped(t, heartbeat)
}

func TestExecutorNormalExitCleansUpProcessTree(t *testing.T) {
	dir := t.TempDir()
	heartbeat := filepath.Join(dir, "heartbeat")
	got, err := executor.NewLocal().Run(context.Background(), helperSpec(t, "spawn-child-exit", heartbeat))
	if err != nil {
		t.Fatalf("run spawn-child-exit: %v", err)
	}
	if got.Code != executor.CodeExit || got.ExitCode == nil || *got.ExitCode != 0 {
		t.Fatalf("observation = %#v, want successful exit", got)
	}
	assertHeartbeatStopped(t, heartbeat)
}

func TestExecutorNormalExitDrainsInheritedPipesAfterProcessTreeCleanup(t *testing.T) {
	dir := t.TempDir()
	heartbeat := filepath.Join(dir, "heartbeat")
	spec := helperSpec(t, "spawn-child-inherit-exit", heartbeat)
	spec.Timeout = 5 * time.Second

	got, err := executor.NewLocal().Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("run spawn-child-inherit-exit: %v; observation=%#v", err, got)
	}
	if got.Code != executor.CodeExit || got.ExitCode == nil || *got.ExitCode != 0 {
		t.Fatalf("observation = %#v, want successful exit", got)
	}
	assertHeartbeatStopped(t, heartbeat)
}

func helperSpec(t *testing.T, mode string, args ...string) config.CommandSpec {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "executor-testhelper")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", exe, "./testhelper")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build test helper: %v\n%s", err, output)
	}
	allArgs := append([]string{mode}, args...)
	return config.CommandSpec{
		ID:             mode,
		Executable:     exe,
		Args:           allArgs,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 64 * 1024,
		Required:       true,
	}
}

func waitForHeartbeat(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("heartbeat file %s was not written", path)
}

func assertHeartbeatStopped(t *testing.T, path string) {
	t.Helper()
	waitForHeartbeat(t, path)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat heartbeat: %v", err)
	}
	size := info.Size()
	time.Sleep(350 * time.Millisecond)
	info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("stat heartbeat after cleanup: %v", err)
	}
	if info.Size() != size {
		t.Fatalf("child process kept writing heartbeat: size grew from %d to %d", size, info.Size())
	}
}
