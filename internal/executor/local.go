package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var secretEnvName = regexp.MustCompile(`(?i)(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)`)

type Local struct {
	BaseEnv            []string
	Clock              func() time.Time
	processTreeFactory func(*exec.Cmd) (processTreeController, error)
	waitCommand        func(*exec.Cmd) error
}

type processTreeController interface {
	afterStart(*exec.Cmd) error
	terminate() error
	close() error
}

func NewLocal() *Local {
	return &Local{}
}

func (l *Local) Run(ctx context.Context, spec config.CommandSpec) (domain.Observation, error) {
	clock := l.Clock
	if clock == nil {
		clock = time.Now
	}
	started := clock()
	runCtx := ctx
	cancel := func() {}
	if spec.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, spec.Timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(runCtx, spec.Executable, spec.Args...)
	if spec.WorkingDirectory != "" {
		cmd.Dir = spec.WorkingDirectory
	}
	cmd.Env = sanitizedEnv(l.BaseEnv)

	processTreeFactory := l.processTreeFactory
	if processTreeFactory == nil {
		processTreeFactory = prepareProcessTree
	}
	controller, err := processTreeFactory(cmd)
	if err != nil {
		return domain.Observation{
			Code:      CodeStartError,
			StartedAt: started,
			EndedAt:   clock(),
			Stderr:    err.Error(),
		}, err
	}
	defer func() { _ = controller.close() }()
	var interrupted atomic.Bool
	cmd.Cancel = func() error {
		interrupted.Store(true)
		_ = controller.terminate()
		return nil
	}
	cmd.WaitDelay = 2 * time.Second

	outCap := spec.MaxOutputBytes
	if outCap <= 0 {
		outCap = 64 * 1024
	}
	outBuf := newBoundedBuffer(outCap)
	errBuf := newBoundedBuffer(outCap)
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return startErrorObservation(started, clock, err), err
	}
	defer stdoutReader.Close()
	defer stdoutWriter.Close()
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		return startErrorObservation(started, clock, err), err
	}
	defer stderrReader.Close()
	defer stderrWriter.Close()
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := cmd.Start(); err != nil {
		return startErrorObservation(started, clock, err), err
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(outBuf, stdoutReader)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(errBuf, stderrReader)
	}()

	if err := controller.afterStart(cmd); err != nil {
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = controller.close()
		wg.Wait()
		return startErrorObservation(started, clock, err), err
	}

	waitCommand := l.waitCommand
	if waitCommand == nil {
		waitCommand = func(cmd *exec.Cmd) error { return cmd.Wait() }
	}
	waitErr := waitCommand(cmd)
	wasInterrupted := interrupted.Load()
	cleanupErr := controller.close()
	drainTruncated := false
	if cleanupErr != nil {
		drainTruncated = true
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
		wg.Wait()
	} else if waitForOutputDrain(runCtx, cmd.WaitDelay, &wg, stdoutReader, stderrReader) {
		drainTruncated = true
		if !wasInterrupted {
			cleanupErr = errors.New("output pipes remained open after process-tree cleanup")
		}
	}

	code := CodeExit
	if wasInterrupted && errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		code = CodeTimeout
	} else if wasInterrupted && errors.Is(runCtx.Err(), context.Canceled) {
		code = CodeCancelled
	}

	exitCode, executionErr := interpretWaitError(waitErr, wasInterrupted)
	if cleanupErr != nil {
		code = CodeCleanupError
		cleanupFailure := fmt.Errorf("%w: %w", ErrProcessCleanup, cleanupErr)
		executionErr = errors.Join(executionErr, cleanupFailure)
	} else if executionErr != nil {
		code = CodeExecutionError
	}
	diagnostic, diagnosticTruncated := executorDiagnostic(code, executionErr)

	return domain.Observation{
		Code:            code,
		ExitCode:        exitCode,
		Stdout:          outBuf.String(),
		Stderr:          errBuf.String(),
		OutputTruncated: outBuf.Truncated() || errBuf.Truncated() || diagnosticTruncated || drainTruncated,
		StartedAt:       started,
		EndedAt:         clock(),
		Data:            diagnostic,
	}, executionErr
}

func waitForOutputDrain(ctx context.Context, maxWait time.Duration, wg *sync.WaitGroup, readers ...io.Closer) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return false
	default:
	}

	timer := time.NewTimer(maxWait)
	defer timer.Stop()
	select {
	case <-done:
		return false
	case <-ctx.Done():
	case <-timer.C:
	}
	for _, reader := range readers {
		_ = reader.Close()
	}
	<-done
	return true
}

func startErrorObservation(started time.Time, clock func() time.Time, err error) domain.Observation {
	return domain.Observation{
		Code:      CodeStartError,
		StartedAt: started,
		EndedAt:   clock(),
		Stderr:    err.Error(),
	}
}

func sanitizedEnv(base []string) []string {
	if base == nil {
		base = os.Environ()
	}
	out := make([]string, 0, len(base))
	for _, entry := range base {
		name, _, ok := strings.Cut(entry, "=")
		if !ok || secretEnvName.MatchString(name) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

type boundedBuffer struct {
	limit     int
	buf       bytes.Buffer
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	accepted := len(p)
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = b.truncated || len(p) > 0
		return accepted, nil
	}
	if remaining > len(p) {
		remaining = len(p)
	}
	_, _ = b.buf.Write(p[:remaining])
	if remaining < len(p) {
		b.truncated = true
	}
	return accepted, nil
}

func (b *boundedBuffer) String() string {
	return b.buf.String()
}

func (b *boundedBuffer) Truncated() bool {
	return b.truncated
}

func interpretWaitError(waitErr error, interrupted bool) (*int, error) {
	if waitErr == nil {
		zero := 0
		return &zero, nil
	}
	if exit, ok := waitErr.(*exec.ExitError); ok {
		code := exit.ExitCode()
		return &code, nil
	}
	if interrupted {
		return nil, nil
	}
	return nil, fmt.Errorf("wait for command: %w", waitErr)
}

const maxExecutorDiagnosticBytes = 768

func executorDiagnostic(code string, err error) (json.RawMessage, bool) {
	if err == nil {
		return nil, false
	}
	message := strings.ToValidUTF8(err.Error(), "\uFFFD")
	truncated := len(message) > maxExecutorDiagnosticBytes
	if truncated {
		message = message[:maxExecutorDiagnosticBytes]
		for len(message) > 0 && !utf8.ValidString(message) {
			message = message[:len(message)-1]
		}
	}
	data, marshalErr := json.Marshal(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    code,
		Message: message,
	})
	if marshalErr != nil {
		return nil, truncated
	}
	return data, truncated
}
