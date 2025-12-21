// Package database provides database client utilities.
package database

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Host        string
	Port        int
	Password    string
	DB          int
	TLSEnabled  bool
	PoolSize    int
	MinIdleConn int
}

// DefaultRedisConfig returns sensible defaults.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Port:        6380, // Azure Redis uses 6380 for TLS
		DB:          0,
		TLSEnabled:  true,
		PoolSize:    100,
		MinIdleConn: 10,
	}
}

// RedisClient wraps the Redis client.
type RedisClient struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisClient creates a new Redis client.
func NewRedisClient(ctx context.Context, config RedisConfig) (*RedisClient, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	opts := &redis.Options{
		Addr:         addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConn,
	}

	if config.TLSEnabled {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client: client,
		config: config,
	}, nil
}

// Client returns the underlying redis client.
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// Ping checks the connection.
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the client.
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// String operations

// Get retrieves a string value.
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	return val, err
}

// Set sets a string value with optional expiration.
func (r *RedisClient) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// SetNX sets a value only if the key doesn't exist (for distributed locks).
func (r *RedisClient) SetNX(ctx context.Context, key, value string, expiration time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// Delete deletes keys.
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Del is an alias for Delete.
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.Delete(ctx, keys...)
}

// Exists checks if keys exist.
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Expire sets an expiration on a key.
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL returns the time to live of a key.
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// JSON operations

// GetJSON retrieves and unmarshals a JSON value.
func (r *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// SetJSON marshals and sets a JSON value.
func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return r.Set(ctx, key, string(data), expiration)
}

// Hash operations

// HGet gets a hash field.
func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := r.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	return val, err
}

// HSet sets hash fields.
func (r *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.HSet(ctx, key, values...).Err()
}

// HGetAll gets all hash fields.
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// HDel deletes hash fields.
func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

// HIncrBy increments a hash field by a specific amount.
func (r *RedisClient) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return r.client.HIncrBy(ctx, key, field, incr).Result()
}

// Geo operations (for location tracking)

// GeoAdd adds a geospatial item.
func (r *RedisClient) GeoAdd(ctx context.Context, key string, locations ...*redis.GeoLocation) error {
	return r.client.GeoAdd(ctx, key, locations...).Err()
}

// GeoPos gets positions of members.
func (r *RedisClient) GeoPos(ctx context.Context, key string, members ...string) ([]*redis.GeoPos, error) {
	return r.client.GeoPos(ctx, key, members...).Result()
}

// GeoRadius finds members within a radius.
func (r *RedisClient) GeoRadius(ctx context.Context, key string, lng, lat float64, query *redis.GeoRadiusQuery) ([]redis.GeoLocation, error) {
	return r.client.GeoRadius(ctx, key, lng, lat, query).Result()
}

// GeoRemove removes a geospatial item.
func (r *RedisClient) GeoRemove(ctx context.Context, key string, members ...string) error {
	return r.client.ZRem(ctx, key, members).Err()
}

// Sorted set operations (for leaderboards, rankings)

// ZAdd adds members to a sorted set.
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}

// ZRange gets members by index range.
func (r *RedisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeByScore gets members by score range.
func (r *RedisClient) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return r.client.ZRangeByScore(ctx, key, opt).Result()
}

// ZRem removes members from a sorted set.
func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.ZRem(ctx, key, members...).Err()
}

// ZRank returns the rank of a member in a sorted set.
func (r *RedisClient) ZRank(ctx context.Context, key, member string) (int64, error) {
	rank, err := r.client.ZRank(ctx, key, member).Result()
	if err == redis.Nil {
		return -1, ErrKeyNotFound
	}
	return rank, err
}

// ZCard returns the number of elements in a sorted set.
func (r *RedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	return r.client.ZCard(ctx, key).Result()
}

// List operations (for queues)

// LPush pushes to the left of a list.
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, key, values...).Err()
}

// RPop pops from the right of a list.
func (r *RedisClient) RPop(ctx context.Context, key string) (string, error) {
	val, err := r.client.RPop(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrKeyNotFound
	}
	return val, err
}

// BRPop blocking pop from the right.
func (r *RedisClient) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return r.client.BRPop(ctx, timeout, keys...).Result()
}

// LLen returns the length of a list.
func (r *RedisClient) LLen(ctx context.Context, key string) (int64, error) {
	return r.client.LLen(ctx, key).Result()
}

// Pub/Sub

// Publish publishes a message to a channel.
func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to channels.
func (r *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// Increment operations

// Incr increments a key.
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// IncrBy increments a key by a specific amount.
func (r *RedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

// IncrByFloat increments a key by a float amount.
func (r *RedisClient) IncrByFloat(ctx context.Context, key string, value float64) (float64, error) {
	return r.client.IncrByFloat(ctx, key, value).Result()
}

// Decr decrements a key.
func (r *RedisClient) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

// Float64 operations

// GetFloat64 retrieves a float64 value.
func (r *RedisClient) GetFloat64(ctx context.Context, key string) (float64, error) {
	val, err := r.client.Get(ctx, key).Float64()
	if err == redis.Nil {
		return 0, ErrKeyNotFound
	}
	return val, err
}

// SetFloat64 sets a float64 value with optional expiration.
func (r *RedisClient) SetFloat64(ctx context.Context, key string, value float64, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Set operations

// SAdd adds members to a set.
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return r.client.SAdd(ctx, key, members...).Result()
}

// SMembers gets all members of a set.
func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// SCard gets the cardinality (size) of a set.
func (r *RedisClient) SCard(ctx context.Context, key string) (int64, error) {
	return r.client.SCard(ctx, key).Result()
}

// SIsMember checks if a member is in the set.
func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// SRem removes members from a set.
func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return r.client.SRem(ctx, key, members...).Result()
}

// Common errors
var ErrKeyNotFound = fmt.Errorf("key not found")

// Distributed Lock

// Lock represents a distributed lock.
type Lock struct {
	client *RedisClient
	key    string
	value  string
	ttl    time.Duration
}

// AcquireLock attempts to acquire a distributed lock.
func (r *RedisClient) AcquireLock(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	value := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := r.SetNX(ctx, key, value, ttl)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, ErrLockNotAcquired
	}
	return &Lock{
		client: r,
		key:    key,
		value:  value,
		ttl:    ttl,
	}, nil
}

// Release releases the lock.
func (l *Lock) Release(ctx context.Context) error {
	// Only release if we still hold the lock
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	return l.client.client.Eval(ctx, script, []string{l.key}, l.value).Err()
}

// Extend extends the lock TTL.
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	return l.client.client.Eval(ctx, script, []string{l.key}, l.value, int64(ttl.Milliseconds())).Err()
}

var ErrLockNotAcquired = fmt.Errorf("lock not acquired")

// Retry-enabled operations for production resilience

// GetWithRetry retrieves a string value with retry logic.
func (r *RedisClient) GetWithRetry(ctx context.Context, key string) (string, error) {
	var result string
	err := RetryRedisOperation(ctx, func() error {
		var getErr error
		result, getErr = r.Get(ctx, key)
		return getErr
	})
	return result, err
}

// SetWithRetry sets a string value with retry logic.
func (r *RedisClient) SetWithRetry(ctx context.Context, key, value string, expiration time.Duration) error {
	return RetryRedisOperation(ctx, func() error {
		return r.Set(ctx, key, value, expiration)
	})
}

// SetNXWithRetry sets a value with retry logic if the key doesn't exist.
func (r *RedisClient) SetNXWithRetry(ctx context.Context, key, value string, expiration time.Duration) (bool, error) {
	var result bool
	err := RetryRedisOperation(ctx, func() error {
		var setErr error
		result, setErr = r.SetNX(ctx, key, value, expiration)
		return setErr
	})
	return result, err
}

// DeleteWithRetry deletes keys with retry logic.
func (r *RedisClient) DeleteWithRetry(ctx context.Context, keys ...string) error {
	return RetryRedisOperation(ctx, func() error {
		return r.Delete(ctx, keys...)
	})
}

// GetJSONWithRetry retrieves and unmarshals a JSON value with retry logic.
func (r *RedisClient) GetJSONWithRetry(ctx context.Context, key string, dest interface{}) error {
	return RetryRedisOperation(ctx, func() error {
		return r.GetJSON(ctx, key, dest)
	})
}

// SetJSONWithRetry marshals and sets a JSON value with retry logic.
func (r *RedisClient) SetJSONWithRetry(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return RetryRedisOperation(ctx, func() error {
		return r.SetJSON(ctx, key, value, expiration)
	})
}

// HGetWithRetry gets a hash field with retry logic.
func (r *RedisClient) HGetWithRetry(ctx context.Context, key, field string) (string, error) {
	var result string
	err := RetryRedisOperation(ctx, func() error {
		var getErr error
		result, getErr = r.HGet(ctx, key, field)
		return getErr
	})
	return result, err
}

// HSetWithRetry sets hash fields with retry logic.
func (r *RedisClient) HSetWithRetry(ctx context.Context, key string, values ...interface{}) error {
	return RetryRedisOperation(ctx, func() error {
		return r.HSet(ctx, key, values...)
	})
}

// HGetAllWithRetry gets all hash fields with retry logic.
func (r *RedisClient) HGetAllWithRetry(ctx context.Context, key string) (map[string]string, error) {
	var result map[string]string
	err := RetryRedisOperation(ctx, func() error {
		var getErr error
		result, getErr = r.HGetAll(ctx, key)
		return getErr
	})
	return result, err
}

// GeoAddWithRetry adds geospatial items with retry logic.
func (r *RedisClient) GeoAddWithRetry(ctx context.Context, key string, locations ...*redis.GeoLocation) error {
	return RetryRedisOperation(ctx, func() error {
		return r.GeoAdd(ctx, key, locations...)
	})
}

// GeoRadiusWithRetry finds members within a radius with retry logic.
func (r *RedisClient) GeoRadiusWithRetry(ctx context.Context, key string, lng, lat float64, query *redis.GeoRadiusQuery) ([]redis.GeoLocation, error) {
	var result []redis.GeoLocation
	err := RetryRedisOperation(ctx, func() error {
		var geoErr error
		result, geoErr = r.GeoRadius(ctx, key, lng, lat, query)
		return geoErr
	})
	return result, err
}

// PublishWithRetry publishes a message with retry logic.
func (r *RedisClient) PublishWithRetry(ctx context.Context, channel string, message interface{}) error {
	return RetryRedisOperation(ctx, func() error {
		return r.Publish(ctx, channel, message)
	})
}

// IncrWithRetry increments a key with retry logic.
func (r *RedisClient) IncrWithRetry(ctx context.Context, key string) (int64, error) {
	var result int64
	err := RetryRedisOperation(ctx, func() error {
		var incrErr error
		result, incrErr = r.Incr(ctx, key)
		return incrErr
	})
	return result, err
}

// AcquireLockWithRetry attempts to acquire a distributed lock with retry logic.
func (r *RedisClient) AcquireLockWithRetry(ctx context.Context, key string, ttl time.Duration, maxAttempts int) (*Lock, error) {
	var lock *Lock
	config := RetryConfig{
		MaxRetries:   maxAttempts - 1,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   1.5,
		Jitter:       0.3,
	}

	err := Retry(ctx, config, func() error {
		var lockErr error
		lock, lockErr = r.AcquireLock(ctx, key, ttl)
		if lockErr == ErrLockNotAcquired {
			return lockErr // Retryable
		}
		return lockErr
	})

	return lock, err
}

// PingWithRetry checks the connection with retry logic.
func (r *RedisClient) PingWithRetry(ctx context.Context) error {
	return RetryRedisOperation(ctx, func() error {
		return r.Ping(ctx)
	})
}
