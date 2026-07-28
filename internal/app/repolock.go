package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var ErrRepoBusy = errors.New("repository already has an active run")

// RepoLocks combines an in-process ownership map with an owner-only,
// create-exclusive lease file. The latter prevents two harness processes from
// operating on the same canonical repository at once.
type RepoLocks struct {
	mu        sync.Mutex
	directory string
	owners    map[string]lease
	keysByRun map[domain.RunID]lease
}

type lease struct {
	key  string
	path string
}

type leaseRecord struct {
	Version   int          `json:"version"`
	Key       string       `json:"key"`
	RunID     domain.RunID `json:"run_id"`
	CreatedAt time.Time    `json:"created_at"`
}

func NewRepoLocks() *RepoLocks {
	directory, err := os.UserCacheDir()
	if err != nil {
		directory = os.TempDir()
	}
	return NewRepoLocksAt(filepath.Join(directory, "ai4se-harness", "repo-locks"))
}

func NewRepoLocksAt(directory string) *RepoLocks {
	return &RepoLocks{
		directory: directory, owners: make(map[string]lease), keysByRun: make(map[domain.RunID]lease),
	}
}

func (l *RepoLocks) Acquire(root string, runID domain.RunID) error {
	key, err := repositoryLockKey(root)
	if err != nil {
		return err
	}
	if runID == "" {
		return ErrRepoBusy
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.owners[key]; exists {
		return ErrRepoBusy
	}
	path := l.leasePath(key)
	if err := os.MkdirAll(l.directory, 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return ErrRepoBusy
	}
	if err != nil {
		return err
	}
	record, marshalErr := json.Marshal(leaseRecord{
		Version: 1, Key: key, RunID: runID, CreatedAt: time.Now().UTC(),
	})
	if marshalErr == nil {
		_, marshalErr = file.Write(record)
	}
	if marshalErr == nil {
		marshalErr = file.Sync()
	}
	closeErr := file.Close()
	if marshalErr != nil || closeErr != nil {
		_ = os.Remove(path)
		if marshalErr != nil {
			return marshalErr
		}
		return closeErr
	}
	entry := lease{key: key, path: path}
	l.owners[key] = entry
	l.keysByRun[runID] = entry
	return nil
}

// Release releases a lease only when the on-disk owner matches runID. Recovery
// calls this method with the durable run snapshot, so it cannot delete a newer
// process's lease for the same repository.
func (l *RepoLocks) Release(root string, runID domain.RunID) {
	key, err := repositoryLockKey(root)
	if err != nil {
		return
	}
	l.release(key, runID)
}

func (l *RepoLocks) ReleaseRun(runID domain.RunID) {
	l.mu.Lock()
	entry, exists := l.keysByRun[runID]
	l.mu.Unlock()
	if exists {
		l.releaseEntry(entry, runID)
		return
	}
	entries, err := os.ReadDir(l.directory)
	if err != nil {
		return
	}
	for _, candidate := range entries {
		if candidate.IsDir() || !strings.HasSuffix(candidate.Name(), ".lease") {
			continue
		}
		path := filepath.Join(l.directory, candidate.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var record leaseRecord
		if json.Unmarshal(data, &record) == nil && record.Version == 1 && record.RunID == runID {
			l.releaseEntry(lease{key: record.Key, path: path}, runID)
			return
		}
	}
}

func (l *RepoLocks) release(key string, runID domain.RunID) {
	l.mu.Lock()
	entry, exists := l.keysByRun[runID]
	l.mu.Unlock()
	if !exists || entry.key != key {
		entry = lease{key: key, path: l.leasePath(key)}
	}
	l.releaseEntry(entry, runID)
}

func (l *RepoLocks) releaseEntry(entry lease, runID domain.RunID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	data, err := os.ReadFile(entry.path)
	if err != nil {
		return
	}
	var record leaseRecord
	if json.Unmarshal(data, &record) != nil || record.Version != 1 || record.Key != entry.key || record.RunID != runID {
		return
	}
	if os.Remove(entry.path) != nil {
		return
	}
	_ = syncLeaseDirectory(l.directory)
	if current, ok := l.owners[entry.key]; ok && current.path == entry.path {
		delete(l.owners, entry.key)
	}
	if current, ok := l.keysByRun[runID]; ok && current.path == entry.path {
		delete(l.keysByRun, runID)
	}
}

func (l *RepoLocks) leasePath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(l.directory, hex.EncodeToString(sum[:])+".lease")
}

func repositoryLockKey(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	key := filepath.Clean(canonical)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key, nil
}
