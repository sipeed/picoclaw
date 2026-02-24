package media

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MediaMeta holds metadata about a stored media file.
type MediaMeta struct {
	Filename    string
	ContentType string
	Source      string // "telegram", "discord", "tool:image-gen", etc.
}

// MediaStore manages the lifecycle of media files associated with processing scopes.
type MediaStore interface {
	// Store registers an existing local file under the given scope.
	// Returns a ref identifier (e.g. "media://<id>").
	// Store does not move or copy the file; it only records the mapping.
	Store(localPath string, meta MediaMeta, scope string) (ref string, err error)

	// Resolve returns the local file path for a given ref.
	Resolve(ref string) (localPath string, err error)

	// ResolveWithMeta returns the local file path and metadata for a given ref.
	ResolveWithMeta(ref string) (localPath string, meta MediaMeta, err error)

	// ReleaseAll deletes all files registered under the given scope
	// and removes the mapping entries. File-not-exist errors are ignored.
	ReleaseAll(scope string) error
}

// mediaEntry holds the path and metadata for a stored media file.
type mediaEntry struct {
	path     string
	meta     MediaMeta
	storedAt time.Time
}

// MediaCleanerConfig configures the background TTL cleanup.
type MediaCleanerConfig struct {
	Enabled  bool
	MaxAge   time.Duration
	Interval time.Duration
}

// FileMediaStore is a pure in-memory implementation of MediaStore.
// Files are expected to already exist on disk (e.g. in /tmp/picoclaw_media/).
type FileMediaStore struct {
	mu          sync.RWMutex
	refs        map[string]mediaEntry
	scopeToRefs map[string]map[string]struct{}
	refToScope  map[string]string

	cleanerCfg MediaCleanerConfig
	stop       chan struct{}
	once       sync.Once
	nowFunc    func() time.Time // for testing
}

// NewFileMediaStore creates a new FileMediaStore without background cleanup.
func NewFileMediaStore() *FileMediaStore {
	return &FileMediaStore{
		refs:        make(map[string]mediaEntry),
		scopeToRefs: make(map[string]map[string]struct{}),
		refToScope:  make(map[string]string),
		nowFunc:     time.Now,
	}
}

// NewFileMediaStoreWithCleanup creates a FileMediaStore with TTL-based background cleanup.
func NewFileMediaStoreWithCleanup(cfg MediaCleanerConfig) *FileMediaStore {
	return &FileMediaStore{
		refs:        make(map[string]mediaEntry),
		scopeToRefs: make(map[string]map[string]struct{}),
		refToScope:  make(map[string]string),
		cleanerCfg:  cfg,
		stop:        make(chan struct{}),
		nowFunc:     time.Now,
	}
}

// Store registers a local file under the given scope. The file must exist.
func (s *FileMediaStore) Store(localPath string, meta MediaMeta, scope string) (string, error) {
	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("media store: %s: %w", localPath, err)
	}

	ref := "media://" + uuid.New().String()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.refs[ref] = mediaEntry{path: localPath, meta: meta, storedAt: s.nowFunc()}
	if s.scopeToRefs[scope] == nil {
		s.scopeToRefs[scope] = make(map[string]struct{})
	}
	s.scopeToRefs[scope][ref] = struct{}{}
	s.refToScope[ref] = scope

	return ref, nil
}

// Resolve returns the local path for the given ref.
func (s *FileMediaStore) Resolve(ref string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.refs[ref]
	if !ok {
		return "", fmt.Errorf("media store: unknown ref: %s", ref)
	}
	return entry.path, nil
}

// ResolveWithMeta returns the local path and metadata for the given ref.
func (s *FileMediaStore) ResolveWithMeta(ref string) (string, MediaMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.refs[ref]
	if !ok {
		return "", MediaMeta{}, fmt.Errorf("media store: unknown ref: %s", ref)
	}
	return entry.path, entry.meta, nil
}

// ReleaseAll removes all files under the given scope and cleans up mappings.
func (s *FileMediaStore) ReleaseAll(scope string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	refs, ok := s.scopeToRefs[scope]
	if !ok {
		return nil
	}

	for ref := range refs {
		if entry, exists := s.refs[ref]; exists {
			if err := os.Remove(entry.path); err != nil && !os.IsNotExist(err) {
				// Log but continue — best effort cleanup
			}
			delete(s.refs, ref)
			delete(s.refToScope, ref)
		}
	}

	delete(s.scopeToRefs, scope)
	return nil
}

// CleanExpired removes all entries older than MaxAge.
// Both the file on disk and the in-memory references are deleted atomically
// under the same mutex, preventing dangling references.
func (s *FileMediaStore) CleanExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := s.nowFunc().Add(-s.cleanerCfg.MaxAge)
	removed := 0

	for ref, entry := range s.refs {
		if entry.storedAt.Before(cutoff) {
			if err := os.Remove(entry.path); err != nil && !os.IsNotExist(err) {
				// Log but continue — best effort cleanup
			}

			scope := s.refToScope[ref]
			if scopeRefs, ok := s.scopeToRefs[scope]; ok {
				delete(scopeRefs, ref)
				if len(scopeRefs) == 0 {
					delete(s.scopeToRefs, scope)
				}
			}

			delete(s.refs, ref)
			delete(s.refToScope, ref)
			removed++
		}
	}

	return removed
}

// Start begins the background cleanup goroutine if cleanup is enabled.
func (s *FileMediaStore) Start() {
	if !s.cleanerCfg.Enabled || s.stop == nil {
		return
	}

	log.Printf("[media] cleanup enabled: interval=%s, max_age=%s",
		s.cleanerCfg.Interval, s.cleanerCfg.MaxAge)

	go func() {
		ticker := time.NewTicker(s.cleanerCfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if n := s.CleanExpired(); n > 0 {
					log.Printf("[media] cleanup: removed %d expired entries", n)
				}
			case <-s.stop:
				return
			}
		}
	}()
}

// Stop terminates the background cleanup goroutine.
func (s *FileMediaStore) Stop() {
	if s.stop == nil {
		return
	}
	s.once.Do(func() {
		close(s.stop)
	})
}
