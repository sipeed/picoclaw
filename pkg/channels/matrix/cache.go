package matrix

import (
	"context"
	"sync"
	"time"
)

type roomKindCacheEntry struct {
	isGroup   bool
	expiresAt time.Time
	touchedAt time.Time
}

type roomKindCache struct {
	mu         sync.Mutex
	entries    map[string]roomKindCacheEntry
	maxEntries int
	ttl        time.Duration
}

func newRoomKindCache(maxEntries int, ttl time.Duration) *roomKindCache {
	if maxEntries <= 0 {
		maxEntries = roomKindCacheMaxEntries
	}
	if ttl <= 0 {
		ttl = roomKindCacheTTL
	}

	return &roomKindCache{
		entries:    make(map[string]roomKindCacheEntry),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

func (c *roomKindCache) get(roomID string, now time.Time) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[roomID]
	if !ok {
		return false, false
	}
	if !entry.expiresAt.After(now) {
		delete(c.entries, roomID)
		return false, false
	}

	return entry.isGroup, true
}

func (c *roomKindCache) set(roomID string, isGroup bool, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[roomID]; ok {
		entry.isGroup = isGroup
		entry.expiresAt = now.Add(c.ttl)
		entry.touchedAt = now
		c.entries[roomID] = entry
		return
	}

	c.cleanupExpiredLocked(now)
	for len(c.entries) >= c.maxEntries {
		if !c.evictOldestLocked() {
			break
		}
	}

	c.entries[roomID] = roomKindCacheEntry{
		isGroup:   isGroup,
		expiresAt: now.Add(c.ttl),
		touchedAt: now,
	}
}

func (c *roomKindCache) cleanupExpired(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cleanupExpiredLocked(now)
}

func (c *roomKindCache) cleanupExpiredLocked(now time.Time) int {
	removed := 0
	for roomID, entry := range c.entries {
		if !entry.expiresAt.After(now) {
			delete(c.entries, roomID)
			removed++
		}
	}
	return removed
}

func (c *roomKindCache) evictOldestLocked() bool {
	if len(c.entries) == 0 {
		return false
	}

	var (
		oldestRoomID string
		oldestAt     time.Time
	)

	for roomID, entry := range c.entries {
		if oldestRoomID == "" || entry.touchedAt.Before(oldestAt) {
			oldestRoomID = roomID
			oldestAt = entry.touchedAt
		}
	}

	delete(c.entries, oldestRoomID)
	return true
}

func (c *MatrixChannel) runRoomKindCacheJanitor(ctx context.Context) {
	ticker := time.NewTicker(roomKindCacheCleanupPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			c.roomKindCache.cleanupExpired(now)
		}
	}
}
