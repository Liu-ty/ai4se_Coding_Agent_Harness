//go:build linux

package executor

import (
	"errors"
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
}

func prepareProcessTree(cmd *exec.Cmd) (processTreeController, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &processTree{}, nil
}

func (p *processTree) afterStart(cmd *exec.Cmd) error {
	p.mu.Lock()
	p.pid = cmd.Process.Pid
	terminate := p.terminatePending && !p.terminated
	if terminate {
		p.terminated = true
	}
	p.mu.Unlock()
	if terminate {
		terminateProcessGroup(cmd.Process.Pid)
	}
	return nil
}

func (p *processTree) terminate() {
	p.mu.Lock()
	if p.terminated {
		p.mu.Unlock()
		return
	}
	if p.pid <= 0 {
		p.terminatePending = true
		p.mu.Unlock()
		return
	}
	p.terminated = true
	pid := p.pid
	p.mu.Unlock()
	terminateProcessGroup(pid)
}

func (p *processTree) close() {
	p.terminate()
}

func terminateProcessGroup(pid int) {
	if err := syscall.Kill(-pid, syscall.SIGTERM); errors.Is(err, syscall.ESRCH) {
		return
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
