package cache

import (
	"context"
)

// Cache is an interface for a key-value cache.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key string, value string)
}

type InMemoryCache struct {
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
	val, ok := c.store[key]
	return val, ok
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value string) error {
	c.store[key] = value
	return nil
}
