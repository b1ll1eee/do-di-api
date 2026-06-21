package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned when a key is not present in the cache.
var ErrCacheMiss = errors.New("cache miss")

// RedisCache is a generic JSON cache backed by Redis.
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache constructs a RedisCache with the given client and default TTL.
func NewRedisCache(client *redis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{client: client, ttl: ttl}
}

// Set serialises value as JSON and stores it under key with the default TTL.
func (c *RedisCache) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redisCache.Set marshal: %w", err)
	}
	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("redisCache.Set redis: %w", err)
	}
	return nil
}

// Get deserialises the cached value into dest.
// Returns ErrCacheMiss when the key does not exist.
func (c *RedisCache) Get(ctx context.Context, key string, dest any) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return fmt.Errorf("redisCache.Get redis: %w", err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("redisCache.Get unmarshal: %w", err)
	}
	return nil
}

// Delete removes the given key from the cache. A missing key is not an error.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redisCache.Delete: %w", err)
	}
	return nil
}

// Ping checks connectivity to Redis.
func (c *RedisCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redisCache.Ping: %w", err)
	}
	return nil
}
