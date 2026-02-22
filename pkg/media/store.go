package media

import (
	"fmt"
	"os"
	"sync"

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

	// ReleaseAll deletes all files registered under the given scope
	// and removes the mapping entries. File-not-exist errors are ignored.
	ReleaseAll(scope string) error
}

// FileMediaStore is a pure in-memory implementation of MediaStore.
// Files are expected to already exist on disk (e.g. in /tmp/picoclaw_media/).
type FileMediaStore struct {
	mu          sync.RWMutex
	refToPath   map[string]string
	scopeToRefs map[string]map[string]struct{}
}

// NewFileMediaStore creates a new FileMediaStore.
func NewFileMediaStore() *FileMediaStore {
	return &FileMediaStore{
		refToPath:   make(map[string]string),
		scopeToRefs: make(map[string]map[string]struct{}),
	}
}

// Store registers a local file under the given scope. The file must exist.
func (s *FileMediaStore) Store(localPath string, meta MediaMeta, scope string) (string, error) {
	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("media store: file does not exist: %s", localPath)
	}

	ref := "media://" + uuid.New().String()[:8]

	s.mu.Lock()
	defer s.mu.Unlock()

	s.refToPath[ref] = localPath
	if s.scopeToRefs[scope] == nil {
		s.scopeToRefs[scope] = make(map[string]struct{})
	}
	s.scopeToRefs[scope][ref] = struct{}{}

	return ref, nil
}

// Resolve returns the local path for the given ref.
func (s *FileMediaStore) Resolve(ref string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path, ok := s.refToPath[ref]
	if !ok {
		return "", fmt.Errorf("media store: unknown ref: %s", ref)
	}
	return path, nil
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
		if path, exists := s.refToPath[ref]; exists {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				// Log but continue — best effort cleanup
			}
			delete(s.refToPath, ref)
		}
	}

	delete(s.scopeToRefs, scope)
	return nil
}
