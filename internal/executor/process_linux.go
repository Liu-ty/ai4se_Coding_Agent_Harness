//go:build linux

package executor

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type processTree struct {
	mu               sync.Mutex
	pid              int
	terminatePending bool
	terminated       bool
	cleanupErr       error
}

func prepareProcessTree(cmd *exec.Cmd) (processTreeController, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &processTree{}, nil
}

func (p *processTree) afterStart(cmd *exec.Cmd) error {
	p.mu.Lock()
	p.pid = cmd.Process.Pid
	terminate := p.terminatePending
	p.mu.Unlock()
	if terminate {
		_ = p.terminate()
	}
	return nil
}

func (p *processTree) terminate() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.terminated {
		return p.cleanupErr
	}
	if p.pid <= 0 {
		p.terminatePending = true
		return nil
	}
	p.terminated = true
	p.cleanupErr = terminateProcessGroup(p.pid)
	return p.cleanupErr
}

func (p *processTree) close() error {
	return p.terminate()
}

func terminateProcessGroup(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return fmt.Errorf("send SIGTERM to process group: %w", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pid, 0); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				return nil
			}
			return fmt.Errorf("check process group after SIGTERM: %w", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("send SIGKILL to process group: %w", err)
	}
	return nil
}
