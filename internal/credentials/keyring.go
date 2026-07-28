package credentials

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
)

const keyringService = "ai4se-harness"

type systemKeyring struct{}

func (systemKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (systemKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (systemKeyring) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

func (systemKeyring) DeleteAll(service string) error {
	return keyring.DeleteAll(service)
}

type KeyringStore struct {
	backend keyring.Keyring
	mu      sync.RWMutex
	updated map[string]time.Time
}

func NewKeyringStore() *KeyringStore {
	return newKeyringStore(systemKeyring{})
}

func newKeyringStore(backend keyring.Keyring) *KeyringStore {
	return &KeyringStore{backend: backend, updated: make(map[string]time.Time)}
}

func (s *KeyringStore) Set(ctx context.Context, ref Ref, secret []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	normalized, err := normalizeRef(ref)
	if err != nil || !validSecret(secret) {
		return ErrInvalidCredential
	}
	if s == nil || s.backend == nil {
		return ErrUnavailable
	}
	if err := s.backend.Set(keyringService, keyringAccount(normalized), string(secret)); err != nil {
		return classifyKeyringError(err)
	}
	if err := s.backend.Set(keyringService, keyringStatusAccount(normalized), "configured"); err != nil {
		rollbackErr := s.backend.Delete(keyringService, keyringAccount(normalized))
		if rollbackErr != nil && !errors.Is(rollbackErr, keyring.ErrNotFound) {
			return errors.Join(classifyKeyringError(err), classifyKeyringError(rollbackErr))
		}
		return classifyKeyringError(err)
	}
	s.mu.Lock()
	s.updated[refIdentity(normalized)] = time.Now().UTC()
	s.mu.Unlock()
	return nil
}

func (s *KeyringStore) Get(ctx context.Context, ref Ref) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	normalized, err := normalizeRef(ref)
	if err != nil {
		return nil, ErrInvalidCredential
	}
	if s == nil || s.backend == nil {
		return nil, ErrUnavailable
	}
	secret, err := s.backend.Get(keyringService, keyringAccount(normalized))
	if err != nil {
		return nil, classifyKeyringError(err)
	}
	return []byte(secret), nil
}

func (s *KeyringStore) Delete(ctx context.Context, ref Ref) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	normalized, err := normalizeRef(ref)
	if err != nil {
		return ErrInvalidCredential
	}
	if s == nil || s.backend == nil {
		return ErrUnavailable
	}
	if err := s.backend.Delete(keyringService, keyringAccount(normalized)); err != nil {
		return classifyKeyringError(err)
	}
	if err := s.backend.Delete(keyringService, keyringStatusAccount(normalized)); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return classifyKeyringError(err)
	}
	s.mu.Lock()
	delete(s.updated, refIdentity(normalized))
	s.mu.Unlock()
	return nil
}

func (s *KeyringStore) Status(ctx context.Context, ref Ref) (Status, error) {
	if err := contextError(ctx); err != nil {
		return Status{}, err
	}
	normalized, err := normalizeRef(ref)
	if err != nil {
		return Status{}, ErrInvalidCredential
	}
	if s == nil || s.backend == nil {
		return Status{}, ErrUnavailable
	}
	_, err = s.backend.Get(keyringService, keyringStatusAccount(normalized))
	err = classifyKeyringError(err)
	if errors.Is(err, ErrNotFound) {
		return Status{Ref: normalized, Backend: "keyring"}, nil
	}
	if err != nil {
		return Status{}, err
	}
	s.mu.RLock()
	updatedAt := s.updated[refIdentity(normalized)]
	s.mu.RUnlock()
	return Status{Ref: normalized, Configured: true, Backend: "keyring", UpdatedAt: updatedAt}, nil
}

func keyringAccount(ref Ref) string {
	return ref.Provider + "|" + ref.Host
}

func keyringStatusAccount(ref Ref) string {
	return keyringAccount(ref) + "|status"
}

func classifyKeyringError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, keyring.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, keyring.ErrSetDataTooBig):
		return ErrInvalidKey
	case errors.Is(err, keyring.ErrUnsupportedPlatform):
		return ErrUnavailable
	}

	return classifyPlatformKeyringError(err)
}

var _ Store = (*KeyringStore)(nil)
