//go:build !linux && !windows

package credentials

func classifyPlatformKeyringError(error) error {
	return ErrKeyring
}
