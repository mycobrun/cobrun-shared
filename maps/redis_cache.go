// Package maps provides a Google Maps Platform adapter.
package maps

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements the Cache interface using Redis.
type RedisCache struct {
	client    redis.UniversalClient
	keyPrefix string
}

// NewRedisCache creates a new Redis cache.
func NewRedisCache(client redis.UniversalClient, keyPrefix string) *RedisCache {
	if keyPrefix == "" {
		keyPrefix = "maps:"
	}
	return &RedisCache{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Get retrieves a cached value.
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.keyPrefix + key
	val, err := c.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}
	return val, nil
}

// Set stores a value in cache with TTL.
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := c.keyPrefix + key
	if err := c.client.Set(ctx, fullKey, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}
	return nil
}

// RedisRateLimiter implements the RateLimiter interface using Redis.
type RedisRateLimiter struct {
	client       redis.UniversalClient
	keyPrefix    string
	limit        int           // requests per window
	window       time.Duration // window size
}

// RateLimiterConfig holds rate limiter configuration.
type RateLimiterConfig struct {
	KeyPrefix string
	Limit     int           // requests per window
	Window    time.Duration // window size
}

// DefaultRateLimiterConfig returns default rate limiter config.
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		KeyPrefix: "maps:ratelimit:",
		Limit:     50,
		Window:    time.Second,
	}
}

// NewRedisRateLimiter creates a new Redis-based rate limiter.
func NewRedisRateLimiter(client redis.UniversalClient, config *RateLimiterConfig) *RedisRateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}
	return &RedisRateLimiter{
		client:    client,
		keyPrefix: config.KeyPrefix,
		limit:     config.Limit,
		window:    config.Window,
	}
}

// Allow checks if a request is allowed without waiting.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) bool {
	fullKey := r.keyPrefix + key

	// Use sliding window rate limiting
	now := time.Now()
	windowStart := now.Add(-r.window).UnixMicro()

	pipe := r.client.Pipeline()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart))

	// Count current entries
	countCmd := pipe.ZCard(ctx, fullKey)

	_, _ = pipe.Exec(ctx)

	count := countCmd.Val()
	return count < int64(r.limit)
}

// Wait waits until the request is allowed or context is cancelled.
func (r *RedisRateLimiter) Wait(ctx context.Context, key string) error {
	fullKey := r.keyPrefix + key

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		now := time.Now()
		windowStart := now.Add(-r.window).UnixMicro()

		// Lua script for atomic rate limiting
		script := `
			local key = KEYS[1]
			local limit = tonumber(ARGV[1])
			local window_start = tonumber(ARGV[2])
			local now = tonumber(ARGV[3])
			local window_ms = tonumber(ARGV[4])
			
			-- Remove old entries
			redis.call('ZREMRANGEBYSCORE', key, '0', window_start)
			
			-- Count current entries
			local count = redis.call('ZCARD', key)
			
			if count < limit then
				-- Add new entry
				redis.call('ZADD', key, now, now)
				redis.call('PEXPIRE', key, window_ms)
				return 0  -- allowed
			else
				-- Get oldest entry to calculate wait time
				local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
				if #oldest >= 2 then
					return oldest[2]  -- return oldest timestamp
				end
				return -1  -- rate limited, no wait time available
			end
		`

		result, err := r.client.Eval(ctx, script, []string{fullKey},
			r.limit,
			windowStart,
			now.UnixMicro(),
			r.window.Milliseconds(),
		).Int64()

		if err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		if result == 0 {
			// Allowed
			return nil
		}

		if result == -1 {
			// Rate limited but can't calculate wait time, use default
			time.Sleep(r.window / time.Duration(r.limit))
			continue
		}

		// Calculate wait time based on oldest entry
		oldestTime := time.UnixMicro(result)
		waitUntil := oldestTime.Add(r.window)
		waitDuration := time.Until(waitUntil)

		if waitDuration > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitDuration):
			}
		}
	}
}

// InMemoryCache implements the Cache interface using in-memory storage.
// Use for testing or single-instance deployments.
type InMemoryCache struct {
	data map[string]cacheEntry
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewInMemoryCache creates a new in-memory cache.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data: make(map[string]cacheEntry),
	}
}

// Get retrieves a cached value.
func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	entry, ok := c.data[key]
	if !ok {
		return nil, nil
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return nil, nil
	}
	return entry.value, nil
}

// Set stores a value in cache with TTL.
func (c *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// NoopRateLimiter is a rate limiter that allows everything.
// Use for testing or when rate limiting is disabled.
type NoopRateLimiter struct{}

// NewNoopRateLimiter creates a new noop rate limiter.
func NewNoopRateLimiter() *NoopRateLimiter {
	return &NoopRateLimiter{}
}

// Allow always returns true.
func (r *NoopRateLimiter) Allow(ctx context.Context, key string) bool {
	return true
}

// Wait always returns immediately.
func (r *NoopRateLimiter) Wait(ctx context.Context, key string) error {
	return nil
}
