package credential

import "sync/atomic"

// SecureStore holds a passphrase in memory.
//
// Uses atomic.Pointer so reads and writes are lock-free.
// The passphrase is never written to disk; callers decide how to
// transport it outside this store (e.g., via cmd.Env or os.Environ).
type SecureStore struct {
	val atomic.Pointer[string]
}

// NewSecureStore creates an empty SecureStore.
func NewSecureStore() *SecureStore {
	return &SecureStore{}
}

// SetString stores the passphrase. An empty string clears the store.
func (s *SecureStore) SetString(passphrase string) {
	if passphrase == "" {
		s.val.Store(nil)
		return
	}
	s.val.Store(&passphrase)
}

// Get returns the stored passphrase, or "" if not set.
func (s *SecureStore) Get() string {
	if p := s.val.Load(); p != nil {
		return *p
	}
	return ""
}

// IsSet reports whether a passphrase is currently stored.
func (s *SecureStore) IsSet() bool {
	return s.val.Load() != nil
}

// Clear removes the stored passphrase.
func (s *SecureStore) Clear() {
	s.val.Store(nil)
}
