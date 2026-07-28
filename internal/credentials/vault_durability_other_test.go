package credentials

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestVaultSyncsParentDirectoryAfterReplaceAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	ref := Ref{Provider: "openai", Host: "api.openai.com"}
	vault := NewVault(path, func() ([]byte, error) { return []byte("password"), nil })
	var calls int
	original := syncVaultDirectory
	syncVaultDirectory = func(string) error { calls++; return nil }
	t.Cleanup(func() { syncVaultDirectory = original })

	if err := vault.Set(context.Background(), ref, []byte("secret")); err != nil {
		t.Fatal(err)
	}
	if err := vault.Delete(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("parent directory sync calls = %d, want 2", calls)
	}
}

func TestVaultReportsCommittedWriteWhenPostReplacePermissionHardeningFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	ref := Ref{Provider: "openai", Host: "api.openai.com"}
	vault := NewVault(path, func() ([]byte, error) { return []byte("password"), nil })
	if err := vault.Set(context.Background(), ref, []byte("old")); err != nil {
		t.Fatal(err)
	}
	syncCalls := 0
	originalSync := syncVaultDirectory
	syncVaultDirectory = func(string) error { syncCalls++; return nil }
	t.Cleanup(func() { syncVaultDirectory = originalSync })
	original := restrictVaultFilePermissions
	restrictVaultFilePermissions = func(path string) error {
		if path == vault.recordPath(ref) {
			return errors.New("injected hardening failure")
		}
		return original(path)
	}
	t.Cleanup(func() { restrictVaultFilePermissions = original })

	err := vault.Set(context.Background(), ref, []byte("new"))
	if !errors.Is(err, ErrVaultCommitted) {
		t.Fatalf("replacement error = %v, want ErrVaultCommitted", err)
	}
	if syncCalls != 1 {
		t.Fatalf("parent directory sync calls before committed permission error = %d, want 1", syncCalls)
	}
	got, getErr := vault.Get(context.Background(), ref)
	if getErr != nil || string(got) != "new" {
		t.Fatalf("persisted replacement = %q, %v", got, getErr)
	}
}
