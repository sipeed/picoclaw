package feishu

import (
	"context"
	"sync"
	"time"
)

// tokenCache implements larkcore.Cache with an extra InvalidateAll method.
// This works around a bug in the Lark SDK v3 where the built-in token retry
// loop does not clear stale tokens from cache on auth errors.
type tokenCache struct {
	mu    sync.RWMutex
	store map[string]*tokenEntry
}

type tokenEntry struct {
	value    string
	expireAt time.Time
}

func newTokenCache() *tokenCache {
	return &tokenCache{store: make(map[string]*tokenEntry)}
}

func (c *tokenCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = &tokenEntry{value: value, expireAt: time.Now().Add(ttl)}
	return nil
}

func (c *tokenCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.store[key]
	if !ok {
		return "", nil
	}
	if e.expireAt.Before(time.Now()) {
		delete(c.store, key)
		return "", nil
	}
	return e.value, nil
}

// InvalidateAll removes all cached tokens, forcing fresh acquisition.
func (c *tokenCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.store)
}
