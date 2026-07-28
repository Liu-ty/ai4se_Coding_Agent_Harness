package credentials_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
)

const (
	canarySecret   = "canary-key"
	masterPassword = "canary-master-password"
)

func TestVaultRoundTripAndWrongPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	vault := credentials.NewVault(path, passwordCallback(masterPassword))

	if err := vault.Set(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}
	got, err := vault.Get(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != canarySecret {
		t.Fatalf("credential = %q", got)
	}
	clear(got)

	if err := vault.Set(context.Background(), ref, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	replaced, err := vault.Get(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if string(replaced) != "replacement" {
		t.Fatalf("replacement credential = %q", replaced)
	}
	clear(replaced)

	bad := credentials.NewVault(path, passwordCallback("wrong-password"))
	if _, err := bad.Get(context.Background(), ref); !errors.Is(err, credentials.ErrDecrypt) {
		t.Fatalf("wrong-password error = %v", err)
	}
}

func TestVaultBindsCredentialToEndpointAndKeepsPlaintextOutOfFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	vault := credentials.NewVault(path, passwordCallback(masterPassword))
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	if err := vault.Set(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}

	if _, err := vault.Get(context.Background(), credentials.Ref{
		Provider: "openai",
		Host:     "gateway.example.test",
	}); !errors.Is(err, credentials.ErrNotFound) {
		t.Fatalf("endpoint-mismatch error = %v", err)
	}
	recordPath := onlyVaultRecord(t, path)
	raw, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, plaintext := range [][]byte{[]byte(canarySecret), []byte(masterPassword)} {
		if bytes.Contains(raw, plaintext) {
			t.Fatal("vault contains plaintext credential material")
		}
	}
	assertOwnerOnlyPermissions(t, recordPath)
}

func TestVaultUsesRequiredFormatAndRejectsUnsafeHeaderBeforeDecrypt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	vault := credentials.NewVault(path, passwordCallback(masterPassword))
	ref := credentials.Ref{Provider: "anthropic", Host: "api.anthropic.com"}
	if err := vault.Set(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}
	recordPath := onlyVaultRecord(t, path)
	raw, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 57 {
		t.Fatalf("vault length = %d", len(raw))
	}
	if string(raw[:8]) != "A4SEVLT1" || raw[8] != 1 {
		t.Fatalf("vault prefix/version = %q/%d", raw[:8], raw[8])
	}
	if got := binary.BigEndian.Uint32(raw[25:29]); got != 3 {
		t.Fatalf("Argon2id time = %d", got)
	}
	if got := binary.BigEndian.Uint32(raw[29:33]); got != 64*1024 {
		t.Fatalf("Argon2id memory = %d KiB", got)
	}
	if raw[33] != 2 || raw[34] != 32 {
		t.Fatalf("Argon2id threads/key length = %d/%d", raw[33], raw[34])
	}

	unsupported := append([]byte(nil), raw...)
	unsupported[8] = 2
	if err := os.WriteFile(recordPath, unsupported, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Get(context.Background(), ref); !errors.Is(err, credentials.ErrUnsupportedVersion) {
		t.Fatalf("unsupported-version error = %v", err)
	}

	unsafe := append([]byte(nil), raw...)
	binary.BigEndian.PutUint32(unsafe[29:33], ^uint32(0))
	if err := os.WriteFile(recordPath, unsafe, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Get(context.Background(), ref); !errors.Is(err, credentials.ErrUnsafeParameters) {
		t.Fatalf("unsafe-parameters error = %v", err)
	}

	weakened := append([]byte(nil), raw...)
	binary.BigEndian.PutUint32(weakened[25:29], 2)
	if err := os.WriteFile(recordPath, weakened, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Get(context.Background(), ref); !errors.Is(err, credentials.ErrUnsafeParameters) {
		t.Fatalf("weakened-parameters error = %v", err)
	}
}

func TestVaultStatusAndDeleteDoNotExposeCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	vault := credentials.NewVault(path, passwordCallback(masterPassword))

	status, err := vault.Status(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if status.Configured {
		t.Fatal("missing vault reported configured")
	}
	if err := vault.Set(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}
	status, err = vault.Status(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Configured || status.Backend != "vault" || status.UpdatedAt.IsZero() {
		t.Fatalf("status = %#v", status)
	}
	if bytes.Contains(raw, []byte(canarySecret)) {
		t.Fatal("vault status contains credential")
	}
	if err := vault.Delete(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Get(context.Background(), ref); !errors.Is(err, credentials.ErrNotFound) {
		t.Fatalf("get-after-delete error = %v", err)
	}
}

func TestVaultStatusDoesNotRequestMasterPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	vault := credentials.NewVault(path, passwordCallback(masterPassword))
	if err := vault.Set(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}

	passwordCalls := 0
	statusOnly := credentials.NewVault(path, func() ([]byte, error) {
		passwordCalls++
		return nil, errors.New("password must not be requested for status")
	})
	status, err := statusOnly.Status(context.Background(), ref)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !status.Configured || status.Backend != "vault" || status.UpdatedAt.IsZero() {
		t.Fatalf("status = %#v", status)
	}
	if passwordCalls != 0 {
		t.Fatalf("password callback calls = %d, want 0", passwordCalls)
	}
}

func TestVaultRepeatedOperationsDoNotMutateBorrowedPasswordBuffer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	password := []byte(masterPassword)
	vault := credentials.NewVault(path, func() ([]byte, error) { return password, nil })

	if err := vault.Set(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Status(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if err := vault.Set(context.Background(), ref, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	got, err := vault.Get(context.Background(), ref)
	if err != nil || string(got) != "replacement" {
		t.Fatalf("replacement = %q, %v", got, err)
	}
	if string(password) != masterPassword {
		t.Fatal("vault mutated password callback's borrowed buffer")
	}
}

func TestVaultStoresMultipleProviderHostReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.bin")
	vault := credentials.NewVault(path, passwordCallback(masterPassword))
	first := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	second := credentials.Ref{Provider: "anthropic", Host: "api.anthropic.com"}
	if err := vault.Set(context.Background(), first, []byte("first-secret")); err != nil {
		t.Fatal(err)
	}
	if err := vault.Set(context.Background(), second, []byte("second-secret")); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		ref  credentials.Ref
		want string
	}{{first, "first-secret"}, {second, "second-secret"}} {
		got, err := vault.Get(context.Background(), test.ref)
		if err != nil || string(got) != test.want {
			t.Fatalf("get %v = %q, %v", test.ref, got, err)
		}
	}
	if err := vault.Delete(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	got, err := vault.Get(context.Background(), second)
	if err != nil || string(got) != "second-secret" {
		t.Fatalf("second credential after first delete = %q, %v", got, err)
	}
}

func passwordCallback(password string) func() ([]byte, error) {
	return func() ([]byte, error) {
		return []byte(password), nil
	}
}

func onlyVaultRecord(t *testing.T, basePath string) string {
	t.Helper()
	matches, err := filepath.Glob(basePath + ".*.vlt")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("vault record count = %d", len(matches))
	}
	return matches[0]
}
