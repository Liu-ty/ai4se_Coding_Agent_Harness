//go:build !linux && !windows

package executor

import "os/exec"

type processTree struct{}

func prepareProcessTree(cmd *exec.Cmd) (*processTree, error) {
	return &processTree{}, nil
}

func (p *processTree) afterStart(cmd *exec.Cmd) error { return nil }
func (p *processTree) terminate()                     {}
func (p *processTree) close()                         {}
