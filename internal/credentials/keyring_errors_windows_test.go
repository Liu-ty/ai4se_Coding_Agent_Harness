//go:build windows

package credentials

import (
	"errors"
	"syscall"
	"testing"
)

func TestWindowsKeyringRawErrorsAreClassifiedByWindowsAdapter(t *testing.T) {
	tests := []struct {
		err  error
		want error
	}{
		{err: syscall.Errno(5), want: ErrLocked},
		{err: syscall.Errno(1312), want: ErrUnavailable},
		{err: syscall.Errno(87), want: ErrInvalidKey},
	}
	for _, test := range tests {
		if got := classifyPlatformKeyringError(test.err); !errors.Is(got, test.want) {
			t.Fatalf("classification of %v = %v", test.err, got)
		}
	}
}
