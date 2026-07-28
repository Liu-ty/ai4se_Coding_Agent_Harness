package credentials

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeyringStoreUsesProviderAndNormalizedHostAsAccountIdentity(t *testing.T) {
	backend := newFakeKeyring()
	store := newKeyringStore(backend)
	ref := Ref{Provider: " OpenAI ", Host: "API.OpenAI.COM"}

	if err := store.Set(context.Background(), ref, []byte(canarySecretInternal)); err != nil {
		t.Fatal(err)
	}
	if _, ok := backend.values[keyringService+"\x00openai|api.openai.com"]; !ok {
		t.Fatalf("keyring credential account was not written: %#v", backend.values)
	}
	got, err := store.Get(context.Background(), Ref{Provider: "openai", Host: "api.openai.com"})
	if err != nil || string(got) != canarySecretInternal {
		t.Fatalf("get = %q, %v", got, err)
	}
}

func TestKeyringErrorsUseAllowlistedFallbackClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "unsupported platform", err: keyring.ErrUnsupportedPlatform, want: ErrUnavailable},
		{name: "oversized invalid key", err: keyring.ErrSetDataTooBig, want: ErrInvalidKey},
		{name: "backend detail is redacted", err: errors.New("backend rejected " + canarySecretInternal), want: ErrKeyring},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := classifyKeyringError(test.err)
			if !errors.Is(got, test.want) {
				t.Fatalf("classification = %v", got)
			}
			if strings.Contains(got.Error(), canarySecretInternal) {
				t.Fatal("classified error contains credential")
			}
			if (errors.Is(got, ErrUnavailable) || errors.Is(got, ErrLocked)) !=
				(errors.Is(test.want, ErrUnavailable) || errors.Is(test.want, ErrLocked)) {
				t.Fatalf("fallback eligibility for %v = %v", test.err, got)
			}
		})
	}
}

func TestKeyringStatusUsesPresenceMetadataInsteadOfRetrievingSecret(t *testing.T) {
	backend := newFakeKeyring()
	store := newKeyringStore(backend)
	ref := Ref{Provider: "openai", Host: "api.openai.com"}
	if err := store.Set(context.Background(), ref, []byte(canarySecretInternal)); err != nil {
		t.Fatal(err)
	}
	backend.getUsers = nil
	status, err := store.Status(context.Background(), ref)
	if err != nil || !status.Configured {
		t.Fatalf("status = %#v, %v", status, err)
	}
	for _, user := range backend.getUsers {
		if user == keyringAccount(ref) {
			t.Fatalf("Status retrieved the plaintext-secret account %q", user)
		}
	}
}

func TestKeyringStatusReturnsUnconfiguredWhenPresenceMetadataIsAbsent(t *testing.T) {
	store := newKeyringStore(newFakeKeyring())
	status, err := store.Status(context.Background(), Ref{Provider: "openai", Host: "api.openai.com"})
	if err != nil || status.Configured {
		t.Fatalf("status = %#v, %v", status, err)
	}
}

func TestKeyringSetRollsBackSecretWhenPresenceMetadataWriteFails(t *testing.T) {
	backend := newFakeKeyring()
	backend.setErrorForSuffix = "|status"
	store := newKeyringStore(backend)
	ref := Ref{Provider: "openai", Host: "api.openai.com"}
	if err := store.Set(context.Background(), ref, []byte(canarySecretInternal)); err == nil {
		t.Fatal("metadata write failure was not returned")
	}
	if _, exists := backend.values[keyringService+"\x00"+keyringAccount(ref)]; exists {
		t.Fatal("secret account remained after metadata write failed")
	}
}

const canarySecretInternal = "canary-key"

type fakeKeyring struct {
	values            map[string]string
	lastService       string
	lastUser          string
	getUsers          []string
	setErrorForSuffix string
}

func newFakeKeyring() *fakeKeyring {
	return &fakeKeyring{values: make(map[string]string)}
}

func (f *fakeKeyring) Set(service, user, password string) error {
	f.lastService, f.lastUser = service, user
	if strings.HasSuffix(user, f.setErrorForSuffix) && f.setErrorForSuffix != "" {
		return errors.New("injected keyring metadata failure")
	}
	f.values[service+"\x00"+user] = password
	return nil
}

func (f *fakeKeyring) Get(service, user string) (string, error) {
	f.lastService, f.lastUser = service, user
	f.getUsers = append(f.getUsers, user)
	value, ok := f.values[service+"\x00"+user]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}

func (f *fakeKeyring) Delete(service, user string) error {
	f.lastService, f.lastUser = service, user
	delete(f.values, service+"\x00"+user)
	return nil
}

func (f *fakeKeyring) DeleteAll(string) error {
	return nil
}

var _ keyring.Keyring = (*fakeKeyring)(nil)
