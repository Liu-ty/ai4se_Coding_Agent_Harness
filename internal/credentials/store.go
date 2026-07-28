package credentials

import (
	"context"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound            = errors.New("credential not found")
	ErrAlreadyConfigured   = errors.New("credential already configured")
	ErrAmbiguousCredential = errors.New("credential exists in multiple stores")
	ErrInvalidCredential   = errors.New("invalid credential")
	ErrInvalidKey          = errors.New("invalid credential key")
	ErrUnavailable         = errors.New("credential store unavailable")
	ErrLocked              = errors.New("credential store locked")
	ErrKeyring             = errors.New("keyring operation failed")
	ErrDecrypt             = errors.New("credential decryption failed")
	ErrInvalidVault        = errors.New("invalid credential vault")
	ErrUnsupportedVersion  = errors.New("unsupported credential vault version")
	ErrUnsafeParameters    = errors.New("unsafe credential vault parameters")
	ErrPasswordUnavailable = errors.New("master password unavailable")
	ErrVaultCommitted      = errors.New("credential vault write committed but permission hardening failed")
)

const maxCredentialBytes = 64 * 1024

var providerIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

type Ref struct {
	Provider string `json:"provider"`
	Host     string `json:"host"`
}

type Status struct {
	Ref        Ref       `json:"ref"`
	Configured bool      `json:"configured"`
	Backend    string    `json:"backend"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Store interface {
	Set(context.Context, Ref, []byte) error
	Get(context.Context, Ref) ([]byte, error)
	Delete(context.Context, Ref) error
	Status(context.Context, Ref) (Status, error)
}

type memoryRecord struct {
	secret    []byte
	updatedAt time.Time
}

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]memoryRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make(map[string]memoryRecord)}
}

func (s *MemoryStore) Set(ctx context.Context, ref Ref, secret []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	normalized, err := normalizeRef(ref)
	if err != nil || !validSecret(secret) {
		return ErrInvalidCredential
	}
	clone := append([]byte(nil), secret...)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records == nil {
		s.records = make(map[string]memoryRecord)
	}
	identity := refIdentity(normalized)
	if previous, ok := s.records[identity]; ok {
		clearBytes(previous.secret)
	}
	s.records[identity] = memoryRecord{secret: clone, updatedAt: time.Now().UTC()}
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, ref Ref) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	normalized, err := normalizeRef(ref)
	if err != nil {
		return nil, ErrInvalidCredential
	}
	s.mu.RLock()
	record, ok := s.records[refIdentity(normalized)]
	var secret []byte
	if ok {
		secret = append([]byte(nil), record.secret...)
	}
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return secret, nil
}

func (s *MemoryStore) Delete(ctx context.Context, ref Ref) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	normalized, err := normalizeRef(ref)
	if err != nil {
		return ErrInvalidCredential
	}
	identity := refIdentity(normalized)
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[identity]
	if !ok {
		return ErrNotFound
	}
	clearBytes(record.secret)
	delete(s.records, identity)
	return nil
}

func (s *MemoryStore) Status(ctx context.Context, ref Ref) (Status, error) {
	if err := contextError(ctx); err != nil {
		return Status{}, err
	}
	normalized, err := normalizeRef(ref)
	if err != nil {
		return Status{}, ErrInvalidCredential
	}
	s.mu.RLock()
	record, configured := s.records[refIdentity(normalized)]
	s.mu.RUnlock()
	return Status{
		Ref:        normalized,
		Configured: configured,
		Backend:    "memory",
		UpdatedAt:  record.updatedAt,
	}, nil
}

func normalizeRef(ref Ref) (Ref, error) {
	provider := strings.ToLower(strings.TrimSpace(ref.Provider))
	if !providerIDPattern.MatchString(provider) {
		return Ref{}, ErrInvalidCredential
	}

	rawHost := strings.TrimSpace(ref.Host)
	parsed, err := url.Parse("//" + rawHost)
	if err != nil || parsed.Scheme != "" || parsed.User != nil || parsed.Host == "" ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Ref{}, ErrInvalidCredential
	}
	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if hostname == "" || strings.ContainsAny(hostname, " \t\r\n") {
		return Ref{}, ErrInvalidCredential
	}
	port := parsed.Port()
	host := hostname
	if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	}
	return Ref{Provider: provider, Host: host}, nil
}

func refIdentity(ref Ref) string {
	return ref.Provider + "\x00" + ref.Host
}

func validSecret(secret []byte) bool {
	return len(secret) > 0 && len(secret) <= maxCredentialBytes
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
