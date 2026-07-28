//go:build !windows

package credentials_test

import (
	"os"
	"testing"
)

func assertOwnerOnlyPermissions(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("vault permissions = %04o", info.Mode().Perm())
	}
}
