//go:build !linux && !windows

package executor

import "os/exec"

func prepareProcessTree(_ *exec.Cmd) (processTreeController, error) {
	return nil, ErrUnsupportedPlatform
}
