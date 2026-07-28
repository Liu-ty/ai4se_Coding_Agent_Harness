package credentials

import (
	"context"
	"errors"
	"sync"
)

type CredentialService interface {
	Add(context.Context, Ref, []byte) error
	Update(context.Context, Ref, []byte) error
	Status(context.Context, Ref) (Status, error)
	Clear(context.Context, Ref) error
	Get(context.Context, string, string) ([]byte, error)
}

type Service struct {
	primary  Store
	fallback Store

	locksMu sync.Mutex
	locks   map[string]*referenceLock
}

type referenceLock struct {
	mu    sync.Mutex
	users int
}

type backendState struct {
	store   Store
	status  Status
	err     error
	present bool
}

func NewService(primary, fallback Store) *Service {
	return &Service{primary: primary, fallback: fallback, locks: make(map[string]*referenceLock)}
}

func (s *Service) Add(ctx context.Context, ref Ref, secret []byte) error {
	normalized, err := normalizeRef(ref)
	if err != nil || !validSecret(secret) {
		return ErrInvalidCredential
	}
	unlock := s.lock(normalized)
	defer unlock()

	primary, fallback := s.inspect(ctx, normalized)
	if err := inspectionError(primary, fallback); err != nil {
		return err
	}
	if primary.status.Configured || fallback.status.Configured {
		return ErrAlreadyConfigured
	}

	copyOfSecret := append([]byte(nil), secret...)
	defer clearBytes(copyOfSecret)
	switch {
	case primary.present && primary.err == nil:
		// Once a keyring write is attempted, any error is ambiguous: it may
		// have committed remotely. Never create a second fallback copy.
		return primary.store.Set(ctx, normalized, copyOfSecret)
	case primary.present && fallbackEligible(primary.err) && fallback.present:
		return fallback.store.Set(ctx, normalized, copyOfSecret)
	case primary.present:
		return primary.err
	case fallback.present:
		return fallback.store.Set(ctx, normalized, copyOfSecret)
	default:
		return ErrUnavailable
	}
}

func (s *Service) Update(ctx context.Context, ref Ref, secret []byte) error {
	normalized, err := normalizeRef(ref)
	if err != nil || !validSecret(secret) {
		return ErrInvalidCredential
	}
	unlock := s.lock(normalized)
	defer unlock()

	primary, fallback := s.inspect(ctx, normalized)
	store, err := configuredBackend(primary, fallback)
	if err != nil {
		return err
	}
	if store == nil {
		return ErrNotFound
	}
	copyOfSecret := append([]byte(nil), secret...)
	defer clearBytes(copyOfSecret)
	return store.Set(ctx, normalized, copyOfSecret)
}

func (s *Service) Status(ctx context.Context, ref Ref) (Status, error) {
	normalized, err := normalizeRef(ref)
	if err != nil {
		return Status{}, ErrInvalidCredential
	}
	unlock := s.lock(normalized)
	defer unlock()

	primary, fallback := s.inspect(ctx, normalized)
	store, err := configuredBackend(primary, fallback)
	if err != nil {
		return Status{}, err
	}
	if store == primary.store && store != nil {
		return primary.status, nil
	}
	if store == fallback.store && store != nil {
		return fallback.status, nil
	}
	if primary.present && primary.err == nil {
		return primary.status, nil
	}
	if fallback.present && fallback.err == nil {
		return fallback.status, nil
	}
	return Status{Ref: normalized}, ErrUnavailable
}

func (s *Service) Clear(ctx context.Context, ref Ref) error {
	normalized, err := normalizeRef(ref)
	if err != nil {
		return ErrInvalidCredential
	}
	unlock := s.lock(normalized)
	defer unlock()

	primary, fallback := s.inspect(ctx, normalized)
	states := []backendState{primary, fallback}
	attempted := false
	var clearErr error
	for _, state := range states {
		if !state.present {
			continue
		}
		if state.err == nil && !state.status.Configured {
			continue
		}
		attempted = true
		err := state.store.Delete(ctx, normalized)
		if err == nil || errors.Is(err, ErrNotFound) {
			continue
		}
		if clearErr == nil {
			clearErr = err
		}
	}
	if clearErr != nil {
		return clearErr
	}
	if !attempted {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Get(ctx context.Context, provider, host string) ([]byte, error) {
	normalized, err := normalizeRef(Ref{Provider: provider, Host: host})
	if err != nil {
		return nil, ErrInvalidCredential
	}
	unlock := s.lock(normalized)
	defer unlock()

	primary, fallback := s.inspect(ctx, normalized)
	store, err := configuredBackend(primary, fallback)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrNotFound
	}
	secret, err := store.Get(ctx, normalized)
	if err != nil {
		return nil, err
	}
	// Store.Get returns owned memory, but clone again at the service port so
	// adapters cannot accidentally expose internal backing storage.
	owned := append([]byte(nil), secret...)
	clearBytes(secret)
	return owned, nil
}

func (s *Service) inspect(ctx context.Context, ref Ref) (backendState, backendState) {
	return inspectStore(ctx, s.primary, ref), inspectStore(ctx, s.fallback, ref)
}

func inspectStore(ctx context.Context, store Store, ref Ref) backendState {
	if store == nil {
		return backendState{}
	}
	status, err := store.Status(ctx, ref)
	return backendState{store: store, status: status, err: err, present: true}
}

func inspectionError(primary, fallback backendState) error {
	if primary.err != nil && !fallbackEligible(primary.err) {
		return primary.err
	}
	if fallback.err != nil {
		return fallback.err
	}
	return nil
}

func configuredBackend(primary, fallback backendState) (Store, error) {
	if err := inspectionError(primary, fallback); err != nil {
		return nil, err
	}
	if primary.status.Configured && fallback.status.Configured {
		return nil, ErrAmbiguousCredential
	}
	if primary.status.Configured {
		return primary.store, nil
	}
	if fallback.status.Configured {
		return fallback.store, nil
	}
	if primary.err != nil {
		return nil, primary.err
	}
	return nil, nil
}

func (s *Service) lock(ref Ref) func() {
	identity := refIdentity(ref)
	s.locksMu.Lock()
	if s.locks == nil {
		s.locks = make(map[string]*referenceLock)
	}
	lock := s.locks[identity]
	if lock == nil {
		lock = &referenceLock{}
		s.locks[identity] = lock
	}
	lock.users++
	s.locksMu.Unlock()
	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		s.locksMu.Lock()
		lock.users--
		if lock.users == 0 && s.locks[identity] == lock {
			delete(s.locks, identity)
		}
		s.locksMu.Unlock()
	}
}

func fallbackEligible(err error) bool {
	return errors.Is(err, ErrUnavailable) || errors.Is(err, ErrLocked)
}

var _ CredentialService = (*Service)(nil)
