//go:build linux

package credentials

import (
	"errors"
	"syscall"
	"testing"

	dbus "github.com/godbus/dbus/v5"
)

func TestLinuxKeyringErrorsExcludeWindowsErrnoMappings(t *testing.T) {
	if got := classifyPlatformKeyringError(syscall.Errno(5)); !errors.Is(got, ErrKeyring) {
		t.Fatalf("Linux errno classification = %v", got)
	}
	if got := classifyPlatformKeyringError(dbus.Error{
		Name: "org.freedesktop.Secret.Error.IsLocked",
	}); !errors.Is(got, ErrLocked) {
		t.Fatalf("Secret Service locked classification = %v", got)
	}
}
