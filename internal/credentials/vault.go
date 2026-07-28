package credentials

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	vaultMagic      = "A4SEVLT1"
	vaultVersion    = byte(1)
	vaultSaltBytes  = 16
	vaultNonceBytes = chacha20poly1305.NonceSizeX
	vaultHeaderSize = 8 + 1 + vaultSaltBytes + 4 + 4 + 1 + 1 + vaultNonceBytes

	argonTime      = uint32(3)
	argonMemoryKiB = uint32(64 * 1024)
	argonThreads   = byte(2)
	argonKeyBytes  = byte(chacha20poly1305.KeySize)
	maxVaultBytes  = vaultHeaderSize + maxCredentialBytes + chacha20poly1305.Overhead
)

// PasswordCallback returns a borrowed password buffer. Vault immediately
// copies it and clears only its owned copy, so callbacks may safely reuse the
// same buffer across operations.
type PasswordCallback func() ([]byte, error)

type Vault struct {
	path     string
	password PasswordCallback
	random   io.Reader
}

var (
	restrictVaultFilePermissions = restrictFilePermissions
	syncVaultDirectory           = syncVaultParent
)

func NewVault(path string, password PasswordCallback) *Vault {
	return &Vault{path: path, password: password, random: rand.Reader}
}

func (v *Vault) Set(ctx context.Context, ref Ref, secret []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	normalized, err := normalizeRef(ref)
	if err != nil || !validSecret(secret) || v == nil || v.path == "" {
		return ErrInvalidCredential
	}
	password, err := v.masterPassword()
	if err != nil {
		return err
	}
	defer clearBytes(password)

	salt := make([]byte, vaultSaltBytes)
	nonce := make([]byte, vaultNonceBytes)
	random := v.random
	if random == nil {
		random = rand.Reader
	}
	if _, err := io.ReadFull(random, salt); err != nil {
		return ErrInvalidVault
	}
	if _, err := io.ReadFull(random, nonce); err != nil {
		return ErrInvalidVault
	}
	derivedKey := argon2.IDKey(password, salt, argonTime, argonMemoryKiB, argonThreads, uint32(argonKeyBytes))
	defer clearBytes(derivedKey)
	aead, err := chacha20poly1305.NewX(derivedKey)
	if err != nil {
		return ErrInvalidVault
	}
	ciphertext := aead.Seal(nil, nonce, secret, associatedData(normalized))

	data := make([]byte, 0, vaultHeaderSize+len(ciphertext))
	data = append(data, vaultMagic...)
	data = append(data, vaultVersion)
	data = append(data, salt...)
	var parameter [4]byte
	binary.BigEndian.PutUint32(parameter[:], argonTime)
	data = append(data, parameter[:]...)
	binary.BigEndian.PutUint32(parameter[:], argonMemoryKiB)
	data = append(data, parameter[:]...)
	data = append(data, argonThreads, argonKeyBytes)
	data = append(data, nonce...)
	data = append(data, ciphertext...)
	return writeVaultAtomically(v.recordPath(normalized), data)
}

func (v *Vault) Get(ctx context.Context, ref Ref) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	normalized, err := normalizeRef(ref)
	if err != nil || v == nil || v.path == "" {
		return nil, ErrInvalidCredential
	}
	data, err := readVault(v.recordPath(normalized))
	if err != nil {
		return nil, err
	}
	parameters, err := parseVault(data)
	if err != nil {
		return nil, err
	}
	password, err := v.masterPassword()
	if err != nil {
		return nil, err
	}
	defer clearBytes(password)
	derivedKey := argon2.IDKey(
		password,
		parameters.salt,
		parameters.time,
		parameters.memory,
		parameters.threads,
		uint32(parameters.keyBytes),
	)
	defer clearBytes(derivedKey)
	aead, err := chacha20poly1305.NewX(derivedKey)
	if err != nil {
		return nil, ErrInvalidVault
	}
	plaintext, err := aead.Open(nil, parameters.nonce, parameters.ciphertext, associatedData(normalized))
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

func (v *Vault) Delete(ctx context.Context, ref Ref) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	normalized, err := normalizeRef(ref)
	if err != nil {
		return ErrInvalidCredential
	}
	path := v.recordPath(normalized)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return ErrInvalidVault
	}
	if err := syncVaultDirectory(filepath.Dir(path)); err != nil {
		return ErrInvalidVault
	}
	return nil
}

func (v *Vault) Status(ctx context.Context, ref Ref) (Status, error) {
	if err := contextError(ctx); err != nil {
		return Status{}, err
	}
	normalized, err := normalizeRef(ref)
	if err != nil || v == nil || v.path == "" {
		return Status{}, ErrInvalidCredential
	}
	info, err := os.Stat(v.recordPath(normalized))
	if errors.Is(err, os.ErrNotExist) {
		return Status{Ref: normalized, Backend: "vault"}, nil
	}
	if err != nil {
		return Status{}, ErrInvalidVault
	}
	return Status{
		Ref:        normalized,
		Configured: true,
		Backend:    "vault",
		UpdatedAt:  info.ModTime().UTC(),
	}, nil
}

type vaultParameters struct {
	salt       []byte
	time       uint32
	memory     uint32
	threads    uint8
	keyBytes   uint8
	nonce      []byte
	ciphertext []byte
}

func parseVault(data []byte) (vaultParameters, error) {
	if len(data) < vaultHeaderSize+chacha20poly1305.Overhead || !bytes.Equal(data[:8], []byte(vaultMagic)) {
		return vaultParameters{}, ErrInvalidVault
	}
	if data[8] != vaultVersion {
		return vaultParameters{}, ErrUnsupportedVersion
	}
	parameters := vaultParameters{
		salt:     data[9:25],
		time:     binary.BigEndian.Uint32(data[25:29]),
		memory:   binary.BigEndian.Uint32(data[29:33]),
		threads:  data[33],
		keyBytes: data[34],
		nonce:    data[35:59],
	}
	if parameters.time != argonTime ||
		parameters.memory != argonMemoryKiB ||
		parameters.threads != argonThreads ||
		parameters.keyBytes != argonKeyBytes {
		return vaultParameters{}, ErrUnsafeParameters
	}
	parameters.ciphertext = data[vaultHeaderSize:]
	return parameters, nil
}

func (v *Vault) masterPassword() ([]byte, error) {
	if v == nil || v.password == nil {
		return nil, ErrPasswordUnavailable
	}
	borrowed, err := v.password()
	if err != nil || len(borrowed) == 0 {
		return nil, ErrPasswordUnavailable
	}
	return append([]byte(nil), borrowed...), nil
}

func readVault(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, ErrInvalidVault
	}
	info, err := file.Stat()
	if err != nil || info.Size() < vaultHeaderSize+chacha20poly1305.Overhead || info.Size() > maxVaultBytes {
		_ = file.Close()
		return nil, ErrInvalidVault
	}
	data := make([]byte, info.Size())
	if _, err := io.ReadFull(file, data); err != nil {
		_ = file.Close()
		return nil, ErrInvalidVault
	}
	if err := file.Close(); err != nil {
		return nil, ErrInvalidVault
	}
	return data, nil
}

func writeVaultAtomically(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return ErrInvalidVault
	}
	temp, err := os.CreateTemp(directory, ".a4se-vault-*")
	if err != nil {
		return ErrInvalidVault
	}
	tempPath := temp.Name()
	keep := false
	defer func() {
		_ = temp.Close()
		if !keep {
			_ = os.Remove(tempPath)
		}
	}()
	if err := restrictVaultFilePermissions(tempPath); err != nil {
		return ErrInvalidVault
	}
	if _, err := temp.Write(data); err != nil {
		return ErrInvalidVault
	}
	if err := temp.Sync(); err != nil {
		return ErrInvalidVault
	}
	if err := temp.Close(); err != nil {
		return ErrInvalidVault
	}
	if err := replaceVaultFile(tempPath, path); err != nil {
		return ErrInvalidVault
	}
	keep = true
	if err := syncVaultDirectory(directory); err != nil {
		return ErrInvalidVault
	}
	if err := restrictVaultFilePermissions(path); err != nil {
		return errors.Join(ErrVaultCommitted, ErrInvalidVault)
	}
	return nil
}

func associatedData(ref Ref) []byte {
	return []byte(refIdentity(ref))
}

func (v *Vault) recordPath(ref Ref) string {
	sum := sha256.Sum256([]byte(refIdentity(ref)))
	return v.path + "." + hex.EncodeToString(sum[:]) + ".vlt"
}

var _ Store = (*Vault)(nil)
