//go:build windows

package credentials

import (
	"errors"
	"syscall"
)

func classifyPlatformKeyringError(err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.Errno(5):
			return ErrLocked
		case syscall.Errno(21), syscall.Errno(1312):
			return ErrUnavailable
		case syscall.Errno(24), syscall.Errno(87):
			return ErrInvalidKey
		}
	}
	return ErrKeyring
}
