//go:build linux

package executor

import (
	"os/exec"
	"syscall"
	"time"
)

type processTree struct {
	pid int
}

func prepareProcessTree(cmd *exec.Cmd) (processTreeController, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &processTree{}, nil
}

func (p *processTree) afterStart(cmd *exec.Cmd) error {
	p.pid = cmd.Process.Pid
	return nil
}

func (p *processTree) terminate() {
	if p.pid <= 0 {
		return
	}
	_ = syscall.Kill(-p.pid, syscall.SIGTERM)
	time.Sleep(2 * time.Second)
	_ = syscall.Kill(-p.pid, syscall.SIGKILL)
}

func (p *processTree) close() {}
