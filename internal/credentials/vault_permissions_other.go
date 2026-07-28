//go:build !windows

package credentials

import "os"

func restrictFilePermissions(path string) error {
	return os.Chmod(path, 0o600)
}
