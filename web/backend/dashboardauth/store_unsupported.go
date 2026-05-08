//go:build mipsle || netbsd || (freebsd && arm)

package dashboardauth

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
)

// Store is unavailable on platforms where modernc sqlite/libc does not build.
type Store struct {
	path string
}

// New reports that the password store is unavailable on this platform.
func New(dir string) (*Store, error) {
	path := filepath.Join(dir, DBFilename)
	s, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	return s, nil
}

// Open reports that the password store is unavailable on this platform.
func Open(path string) (*Store, error) {
	return nil, unsupportedPlatformError()
}

// Close is a no-op for unsupported platforms.
func (s *Store) Close() error { return nil }

// DBPath returns the configured path, if any.
func (s *Store) DBPath() string {
	if s == nil {
		return ""
	}
	return s.path
}

// IsInitialized reports that the store is unavailable on this platform.
func (s *Store) IsInitialized(context.Context) (bool, error) {
	return false, unsupportedPlatformError()
}

// SetPassword reports that the store is unavailable on this platform.
func (s *Store) SetPassword(context.Context, string) error {
	return unsupportedPlatformError()
}

// VerifyPassword reports that the store is unavailable on this platform.
func (s *Store) VerifyPassword(context.Context, string) (bool, error) {
	return false, unsupportedPlatformError()
}

func unsupportedPlatformError() error {
	return fmt.Errorf("%w (%s/%s)", ErrUnsupportedPlatform, runtime.GOOS, runtime.GOARCH)
}