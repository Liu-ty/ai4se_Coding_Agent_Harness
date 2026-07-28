//go:build !windows

package credentials

import "os"

func replaceVaultFile(replacement, target string) error {
	return os.Rename(replacement, target)
}
