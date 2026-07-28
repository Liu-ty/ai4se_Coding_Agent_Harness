//go:build linux

package credentials

import (
	"errors"
	"os/exec"
	"strings"

	dbus "github.com/godbus/dbus/v5"
)

func classifyPlatformKeyringError(err error) error {
	var dbusErr dbus.Error
	if errors.As(err, &dbusErr) {
		switch dbusErr.Name {
		case "org.freedesktop.Secret.Error.IsLocked":
			return ErrLocked
		case "org.freedesktop.DBus.Error.ServiceUnknown",
			"org.freedesktop.DBus.Error.NameHasNoOwner",
			"org.freedesktop.DBus.Error.NoReply",
			"org.freedesktop.DBus.Error.Disconnected",
			"org.freedesktop.DBus.Error.NotSupported":
			return ErrUnavailable
		default:
			return ErrKeyring
		}
	}

	var missingExecutable *exec.Error
	if errors.As(err, &missingExecutable) && missingExecutable.Name == "dbus-launch" {
		return ErrUnavailable
	}
	switch message := err.Error(); {
	case message == "dbus: couldn't determine address of session bus":
		return ErrUnavailable
	case strings.HasPrefix(message, "failed to unlock correct collection "):
		return ErrLocked
	default:
		return ErrKeyring
	}
}
