package providers

import (
	"sync"
)

// KeyRotator provides thread-safe round-robin API key selection.
type KeyRotator struct {
	keys  []string
	index uint64
	mu    sync.Mutex
}

// NewKeyRotator creates a new KeyRotator with the given keys.
func NewKeyRotator(keys []string) *KeyRotator {
	return &KeyRotator{
		keys: keys,
	}
}

// Len returns the number of API keys.
func (kr *KeyRotator) Len() int {
	return len(kr.keys)
}

// Next returns the next API key in round-robin order.
func (kr *KeyRotator) Next() string {
	kr.mu.Lock()
	defer kr.mu.Unlock()
	key := kr.keys[kr.index%uint64(len(kr.keys))]
	kr.index++
	return key
}
