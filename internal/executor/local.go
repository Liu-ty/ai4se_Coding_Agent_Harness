package executor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var secretEnvName = regexp.MustCompile(`(?i)(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)`)

type Local struct {
	BaseEnv []string
	Clock   func() time.Time
}

type processTreeController interface {
	afterStart(*exec.Cmd) error
	terminate()
	close()
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

	controller, err := prepareProcessTree(cmd)
	if err != nil {
		return domain.Observation{
			Code:      CodeStartError,
			StartedAt: started,
			EndedAt:   clock(),
			Stderr:    err.Error(),
		}, err
	}
	defer controller.close()
	cmd.Cancel = func() error {
		controller.terminate()
		return nil
	}
	cmd.WaitDelay = 2 * time.Second

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return startErrorObservation(started, clock, err), err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return startErrorObservation(started, clock, err), err
	}
	if err := cmd.Start(); err != nil {
		return startErrorObservation(started, clock, err), err
	}
	if err := controller.afterStart(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return startErrorObservation(started, clock, err), err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	outCap := spec.MaxOutputBytes
	if outCap <= 0 {
		outCap = 64 * 1024
	}
	outBuf := newBoundedBuffer(outCap)
	errBuf := newBoundedBuffer(outCap)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(outBuf, stdout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(errBuf, stderr)
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	code := CodeExit
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		code = CodeTimeout
	} else if errors.Is(ctx.Err(), context.Canceled) {
		code = CodeCancelled
	}

	var exitCode *int
	if waitErr == nil {
		zero := 0
		exitCode = &zero
	} else if exit, ok := waitErr.(*exec.ExitError); ok {
		codeValue := exit.ExitCode()
		exitCode = &codeValue
	}

	return domain.Observation{
		Code:            code,
		ExitCode:        exitCode,
		Stdout:          outBuf.String(),
		Stderr:          errBuf.String(),
		OutputTruncated: outBuf.Truncated() || errBuf.Truncated(),
		StartedAt:       started,
		EndedAt:         clock(),
	}, nil
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
