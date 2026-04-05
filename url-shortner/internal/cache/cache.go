package cache

import (
	"context"
	"sync"
)

// Cache is an interface for a key-value cache.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key string, value string)
}

type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		store: make(map[string]string),
	}
}

// Get retrieves the value for the given key from the cache.
// If the key is not found, it returns an empty string and false.
func (c *InMemoryCache) Get(ctx context.Context, key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.store[key]
	return val, ok
}

// Set stores the key-value pair in the cache.
func (c *InMemoryCache) Set(ctx context.Context, key string, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
	return nil
}
