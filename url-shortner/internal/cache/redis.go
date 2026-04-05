package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is a redis-based cache implementation of the Cache interface.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new RedisCache with the given address.
func NewRedisCache(addr string) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisCache{client: rdb}
}

// Get retrieves the value for the given key.
// Returns the value and true if the key exists, otherwise returns an empty string and false.
func (c *RedisCache) Get(ctx context.Context, key string) (string, bool) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false
	}
	return val, true
}

// Set sets the value for the given key with no ttl (yet).
func (c *RedisCache) Set(ctx context.Context, key string, value string) {
	c.client.Set(ctx, key, value, 5*time.Minute) // no ttl yet
}
