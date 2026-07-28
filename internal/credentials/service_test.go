package credentials_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
)

func TestStatusNeverContainsSecret(t *testing.T) {
	service := credentials.NewService(credentials.NewMemoryStore(), nil)
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	if err := service.Add(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}

	status, err := service.Status(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(canarySecret)) {
		t.Fatalf("status leaked credential: %s", raw)
	}
	if !status.Configured {
		t.Fatal("status does not report configured credential")
	}
}

func TestServiceAddUpdateGetClearAndNormalizesIdentity(t *testing.T) {
	service := credentials.NewService(credentials.NewMemoryStore(), nil)
	ctx := context.Background()
	ref := credentials.Ref{Provider: " OpenAI ", Host: "API.OpenAI.COM"}
	secret := []byte(canarySecret)

	if err := service.Add(ctx, ref, secret); err != nil {
		t.Fatal(err)
	}
	secret[0] = 'X'
	if err := service.Add(ctx, ref, []byte("duplicate")); !errors.Is(err, credentials.ErrAlreadyConfigured) {
		t.Fatalf("duplicate-add error = %v", err)
	}
	got, err := service.Get(ctx, "openai", "api.openai.com")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != canarySecret {
		t.Fatalf("stored credential = %q", got)
	}
	got[0] = 'Y'
	again, err := service.Get(ctx, "OPENAI", "API.OPENAI.COM")
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != canarySecret {
		t.Fatalf("returned credential aliases store memory: %q", again)
	}

	if err := service.Update(ctx, ref, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	updated, err := service.Get(ctx, "openai", "api.openai.com")
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != "replacement" {
		t.Fatalf("updated credential = %q", updated)
	}
	if _, err := service.Get(ctx, "openai", "gateway.example.test"); !errors.Is(err, credentials.ErrNotFound) {
		t.Fatalf("endpoint-mismatch error = %v", err)
	}

	if err := service.Clear(ctx, ref); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if status.Configured {
		t.Fatal("cleared credential remains configured")
	}
	if _, err := service.Get(ctx, "openai", "api.openai.com"); !errors.Is(err, credentials.ErrNotFound) {
		t.Fatalf("get-after-clear error = %v", err)
	}
}

func TestServiceFallsBackOnlyForUnavailableOrLockedPrimary(t *testing.T) {
	for _, primaryErr := range []error{credentials.ErrUnavailable, credentials.ErrLocked} {
		t.Run(primaryErr.Error(), func(t *testing.T) {
			fallback := credentials.NewMemoryStore()
			service := credentials.NewService(&errorStore{err: primaryErr}, fallback)
			ref := credentials.Ref{Provider: "anthropic", Host: "api.anthropic.com"}
			if err := service.Add(context.Background(), ref, []byte(canarySecret)); err != nil {
				t.Fatal(err)
			}
			status, err := service.Status(context.Background(), ref)
			if err != nil {
				t.Fatal(err)
			}
			if !status.Configured || status.Backend != "memory" {
				t.Fatalf("fallback status = %#v", status)
			}
			got, err := service.Get(context.Background(), ref.Provider, ref.Host)
			if err != nil || string(got) != canarySecret {
				t.Fatalf("fallback get = %q, %v", got, err)
			}
		})
	}
}

func TestServiceDoesNotFallbackForInvalidCredential(t *testing.T) {
	fallback := credentials.NewMemoryStore()
	service := credentials.NewService(&errorStore{err: credentials.ErrInvalidKey}, fallback)
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}

	err := service.Add(context.Background(), ref, []byte(canarySecret))
	if !errors.Is(err, credentials.ErrInvalidKey) {
		t.Fatalf("add error = %v", err)
	}
	status, statusErr := fallback.Status(context.Background(), ref)
	if statusErr != nil {
		t.Fatal(statusErr)
	}
	if status.Configured {
		t.Fatal("invalid credential was written to fallback")
	}
	if bytes.Contains([]byte(err.Error()), []byte(canarySecret)) {
		t.Fatal("error contains credential")
	}
}

func TestServiceStatusReturnsUnavailableWhenNoFallbackExists(t *testing.T) {
	service := credentials.NewService(&errorStore{err: credentials.ErrUnavailable}, nil)
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	if _, err := service.Status(context.Background(), ref); !errors.Is(err, credentials.ErrUnavailable) {
		t.Fatalf("status error = %v", err)
	}
}

func TestServiceDoesNotFallbackAfterAmbiguousPrimaryWrite(t *testing.T) {
	primaryMemory := credentials.NewMemoryStore()
	primary := &writeThenErrorStore{Store: primaryMemory, err: credentials.ErrUnavailable}
	fallback := credentials.NewMemoryStore()
	service := credentials.NewService(primary, fallback)
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}

	if err := service.Add(context.Background(), ref, []byte(canarySecret)); !errors.Is(err, credentials.ErrUnavailable) {
		t.Fatalf("add error = %v", err)
	}
	status, err := fallback.Status(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if status.Configured {
		t.Fatal("ambiguous primary write was copied to fallback")
	}
}

func TestServiceRejectsSplitBrainAndClearDeletesBothCopies(t *testing.T) {
	primary := credentials.NewMemoryStore()
	fallback := credentials.NewMemoryStore()
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	if err := primary.Set(context.Background(), ref, []byte("stale-primary")); err != nil {
		t.Fatal(err)
	}
	if err := fallback.Set(context.Background(), ref, []byte("current-fallback")); err != nil {
		t.Fatal(err)
	}
	service := credentials.NewService(primary, fallback)

	if _, err := service.Get(context.Background(), ref.Provider, ref.Host); !errors.Is(err, credentials.ErrAmbiguousCredential) {
		t.Fatalf("split-brain get error = %v", err)
	}
	if err := service.Update(context.Background(), ref, []byte("replacement")); !errors.Is(err, credentials.ErrAmbiguousCredential) {
		t.Fatalf("split-brain update error = %v", err)
	}
	if err := service.Clear(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	for name, store := range map[string]credentials.Store{"primary": primary, "fallback": fallback} {
		status, err := store.Status(context.Background(), ref)
		if err != nil {
			t.Fatal(err)
		}
		if status.Configured {
			t.Fatalf("%s copy survived clear", name)
		}
	}
}

func TestServiceClearReportsUnknownPrimaryAfterClearingFallback(t *testing.T) {
	primary := &errorStore{err: credentials.ErrUnavailable}
	fallback := credentials.NewMemoryStore()
	ref := credentials.Ref{Provider: "anthropic", Host: "api.anthropic.com"}
	if err := fallback.Set(context.Background(), ref, []byte(canarySecret)); err != nil {
		t.Fatal(err)
	}
	service := credentials.NewService(primary, fallback)

	if err := service.Clear(context.Background(), ref); !errors.Is(err, credentials.ErrUnavailable) {
		t.Fatalf("clear error = %v", err)
	}
	status, err := fallback.Status(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if status.Configured {
		t.Fatal("known fallback copy survived clear")
	}
}

func TestServiceAddIsAtomicPerCredentialReference(t *testing.T) {
	store := newCoordinatedAddStore()
	service := credentials.NewService(store, nil)
	ref := credentials.Ref{Provider: "openai", Host: "api.openai.com"}
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			results <- service.Add(context.Background(), ref, []byte(canarySecret))
		}()
	}
	close(start)

	var added, duplicate int
	for i := 0; i < 2; i++ {
		switch err := <-results; {
		case err == nil:
			added++
		case errors.Is(err, credentials.ErrAlreadyConfigured):
			duplicate++
		default:
			t.Fatalf("add error = %v", err)
		}
	}
	if added != 1 || duplicate != 1 || store.setCalls.Load() != 1 {
		t.Fatalf("added=%d duplicate=%d set calls=%d", added, duplicate, store.setCalls.Load())
	}
}

func TestServiceRejectsInvalidReferencesAndEmptyKeys(t *testing.T) {
	service := credentials.NewService(credentials.NewMemoryStore(), nil)
	tests := []struct {
		name string
		ref  credentials.Ref
		key  []byte
	}{
		{name: "empty provider", ref: credentials.Ref{Host: "api.openai.com"}, key: []byte("x")},
		{name: "URL instead of host", ref: credentials.Ref{Provider: "openai", Host: "https://api.openai.com"}, key: []byte("x")},
		{name: "path in host", ref: credentials.Ref{Provider: "openai", Host: "api.openai.com/v1"}, key: []byte("x")},
		{name: "empty key", ref: credentials.Ref{Provider: "openai", Host: "api.openai.com"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := service.Add(context.Background(), test.ref, test.key); !errors.Is(err, credentials.ErrInvalidCredential) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

type errorStore struct {
	err error
}

func (s *errorStore) Set(context.Context, credentials.Ref, []byte) error {
	return s.err
}

func (s *errorStore) Get(context.Context, credentials.Ref) ([]byte, error) {
	return nil, s.err
}

func (s *errorStore) Delete(context.Context, credentials.Ref) error {
	return s.err
}

func (s *errorStore) Status(context.Context, credentials.Ref) (credentials.Status, error) {
	return credentials.Status{}, s.err
}

var _ credentials.Store = (*errorStore)(nil)

type writeThenErrorStore struct {
	credentials.Store
	err error
}

func (s *writeThenErrorStore) Set(ctx context.Context, ref credentials.Ref, secret []byte) error {
	if err := s.Store.Set(ctx, ref, secret); err != nil {
		return err
	}
	return s.err
}

type coordinatedAddStore struct {
	*credentials.MemoryStore
	statusCalls atomic.Int32
	setCalls    atomic.Int32
	second      chan struct{}
	once        sync.Once
}

func newCoordinatedAddStore() *coordinatedAddStore {
	return &coordinatedAddStore{
		MemoryStore: credentials.NewMemoryStore(),
		second:      make(chan struct{}),
	}
}

func (s *coordinatedAddStore) Status(ctx context.Context, ref credentials.Ref) (credentials.Status, error) {
	status, err := s.MemoryStore.Status(ctx, ref)
	if err != nil || status.Configured {
		return status, err
	}
	call := s.statusCalls.Add(1)
	if call == 2 {
		s.once.Do(func() { close(s.second) })
	} else {
		select {
		case <-s.second:
		case <-time.After(200 * time.Millisecond):
		}
	}
	return status, nil
}

func (s *coordinatedAddStore) Set(ctx context.Context, ref credentials.Ref, secret []byte) error {
	s.setCalls.Add(1)
	return s.MemoryStore.Set(ctx, ref, secret)
}
