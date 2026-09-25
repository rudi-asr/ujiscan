// Package cache provides in-memory caching with TTL
package cache

import (
	"sync"
	"time"
)

// Entry represents a cached item with TTL
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// Cache is a thread-safe in-memory cache with TTL
type Cache struct {
	mu      sync.RWMutex
	items   map[string]*Entry
	maxSize int // Maximum number of items (LRU eviction)
}

// New creates a new cache with max size
func New(maxSize int) *Cache {
	return &Cache{
		items:   make(map[string]*Entry),
		maxSize: maxSize,
	}
}

// Set adds or updates an item with TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
	if len(c.items) >= c.maxSize && c.items[key] == nil {
		c.evictOne()
	}

	c.items[key] = &Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves an item if it exists and hasn't expired
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Value, true
}

// Delete removes an item
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*Entry)
}

// Size returns number of non-expired items
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	now := time.Now()
	for _, entry := range c.items {
		if now.Before(entry.ExpiresAt) {
			count++
		}
	}
	return count
}

// evictOne removes the oldest expired entry, or random entry if none expired
func (c *Cache) evictOne() {
	// First pass: remove expired entries
	var oldestKey string
	var oldestExpiry time.Time

	for key, entry := range c.items {
		if time.Now().After(entry.ExpiresAt) {
			delete(c.items, key)
			return
		}

		// Track oldest non-expired entry
		if oldestExpiry.IsZero() || entry.ExpiresAt.Before(oldestExpiry) {
			oldestKey = key
			oldestExpiry = entry.ExpiresAt
		}
	}

	// If no expired entries, evict the one expiring soonest (LRU-like)
	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// Cleanup removes all expired entries (should be called periodically)
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	evicted := 0

	for key, entry := range c.items {
		if now.After(entry.ExpiresAt) {
			delete(c.items, key)
			evicted++
		}
	}

	return evicted
}
