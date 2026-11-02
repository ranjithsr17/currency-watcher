// Package cache provides a small generic in-memory cache with a fixed TTL.
package cache

import (
	"sync"
	"time"
)

// Entry is a cached value and the times that define its freshness.
type Entry[V any] struct {
	Value     V
	StoredAt  time.Time
	ExpiresAt time.Time
}

// Cache is a map guarded by a sync.RWMutex, so it is safe for concurrent use.
//
// Expired entries are ignored on read and replaced by the next Set. There is no
// background sweep because callers only store validated keys, which keeps the
// number of entries bounded.
type Cache[V any] struct {
	mu    sync.RWMutex
	items map[string]Entry[V]
	ttl   time.Duration
	now   func() time.Time
}

// New returns an empty cache. The now function is injected so that tests can
// control expiry without sleeping.
func New[V any](ttl time.Duration, now func() time.Time) *Cache[V] {
	return &Cache[V]{
		items: make(map[string]Entry[V]),
		ttl:   ttl,
		now:   now,
	}
}

// Get returns the entry for key if it exists and has not expired.
func (c *Cache[V]) Get(key string) (Entry[V], bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || !c.now().Before(entry.ExpiresAt) {
		return Entry[V]{}, false
	}
	return entry, true
}

// Set stores value under key for the cache's TTL and returns the new entry.
func (c *Cache[V]) Set(key string, value V) Entry[V] {
	now := c.now()
	entry := Entry[V]{
		Value:     value,
		StoredAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	c.mu.Lock()
	c.items[key] = entry
	c.mu.Unlock()

	return entry
}
