//go:build windows

package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsVaultReplacementReplacesExistingFile(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.vlt")
	replacement := filepath.Join(directory, "replacement.vlt")
	if err := os.WriteFile(target, []byte("old ciphertext"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("new ciphertext"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceVaultFile(replacement, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new ciphertext" {
		t.Fatalf("replacement content = %q", got)
	}
	if _, err := os.Stat(replacement); !os.IsNotExist(err) {
		t.Fatalf("replacement source still exists: %v", err)
	}
}
