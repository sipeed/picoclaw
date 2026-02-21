package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type registryEntry struct {
	ContainerName string `json:"container_name"`
	Image         string `json:"image"`
	ConfigHash    string `json:"config_hash"`
	CreatedAtMs   int64  `json:"created_at_ms"`
	LastUsedAtMs  int64  `json:"last_used_at_ms"`
}

type registryData struct {
	Entries []registryEntry `json:"entries"`
}

var registryMu sync.Mutex

const registryLockTimeout = 3 * time.Second

type registryFileLock struct {
	path string
}

func acquireRegistryFileLock(registryPath string) (*registryFileLock, error) {
	lockPath := registryPath + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(registryLockTimeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_ = f.Close()
			return &registryFileLock{path: lockPath}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout acquiring registry lock: %s", lockPath)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (l *registryFileLock) release() {
	if l == nil || l.path == "" {
		return
	}
	_ = os.Remove(l.path)
}

func computeConfigHash(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func loadRegistry(path string) (*registryData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &registryData{Entries: []registryEntry{}}, nil
		}
		return nil, err
	}
	var data registryData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if data.Entries == nil {
		data.Entries = []registryEntry{}
	}
	return &data, nil
}

func saveRegistry(path string, data *registryData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(raw, '\n'), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func upsertRegistryEntry(path string, entry registryEntry) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	lock, err := acquireRegistryFileLock(path)
	if err != nil {
		return err
	}
	defer lock.release()

	data, err := loadRegistry(path)
	if err != nil {
		return err
	}

	replaced := false
	for i := range data.Entries {
		if data.Entries[i].ContainerName == entry.ContainerName {
			createdAt := data.Entries[i].CreatedAtMs
			if createdAt > 0 {
				entry.CreatedAtMs = createdAt
			}
			data.Entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		data.Entries = append(data.Entries, entry)
	}

	return saveRegistry(path, data)
}

func removeRegistryEntry(path, containerName string) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	lock, err := acquireRegistryFileLock(path)
	if err != nil {
		return err
	}
	defer lock.release()

	data, err := loadRegistry(path)
	if err != nil {
		return err
	}
	next := make([]registryEntry, 0, len(data.Entries))
	for _, e := range data.Entries {
		if e.ContainerName != containerName {
			next = append(next, e)
		}
	}
	data.Entries = next
	return saveRegistry(path, data)
}
