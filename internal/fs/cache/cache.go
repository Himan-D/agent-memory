package cache

import (
	"sync"
	"time"
)

// Cache implements a simple in-memory cache with TTL
// Pattern: follows cache/embedding.go pool pattern
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	maxSize  int
	cleanups int64
}

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value   interface{}
	Expires time.Time
	Size    int
}

// NewCache creates a new cache
// Pattern: like NewEmbeddingCache() in cache/embedding.go
func NewCache(maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = 10000 // Default 10K entries
	}
	return &Cache{
		items:   make(map[string]*CacheItem),
		maxSize: maxSize,
	}
}

// Get retrieves an item from cache
// Returns (value, true) if found and not expired
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.Expires) {
		// Lazy cleanup
		go c.Delete(key)
		return nil, false
	}

	return item.Value, true
}

// Set stores an item in cache with TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check size limit
	if len(c.items) >= c.maxSize {
		// Simple eviction: delete oldest 10%
		go c.evictOldest()
	}

	size := estimateSize(value)
	c.items[key] = &CacheItem{
		Value:   value,
		Expires: time.Now().Add(ttl),
		Size:    size,
	}
}

// Delete removes an item from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
}

// evictOldest removes 10% of oldest items
// Simplified: just delete random entries
func (c *Cache) evictOldest() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple strategy: clear 10% when full
	toDelete := len(c.items) / 10
	if toDelete < 1 {
		toDelete = 1
	}

	count := 0
	for key := range c.items {
		delete(c.items, key)
		count++
		if count >= toDelete {
			break
		}
	}
	c.cleanups++
}

// Size returns the number of items in cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// estimateSize estimates the memory size of a value
func estimateSize(value interface{}) int {
	if s, ok := value.([]byte); ok {
		return len(s)
	}
	if s, ok := value.(string); ok {
		return len(s)
	}
	return 1024 // Default estimate: 1KB
}
