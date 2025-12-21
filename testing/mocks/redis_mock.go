// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"sync"
	"time"
)

// MockRedisClient is a mock implementation of Redis for testing.
type MockRedisClient struct {
	mu       sync.RWMutex
	data     map[string]string
	expiry   map[string]time.Time
	sets     map[string]map[string]struct{}
	zsets    map[string]map[string]float64
	counters map[string]int64
}

// NewMockRedisClient creates a new mock Redis client.
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data:     make(map[string]string),
		expiry:   make(map[string]time.Time),
		sets:     make(map[string]map[string]struct{}),
		zsets:    make(map[string]map[string]float64),
		counters: make(map[string]int64),
	}
}

// Get retrieves a value by key.
func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check expiry
	if exp, ok := m.expiry[key]; ok && time.Now().After(exp) {
		delete(m.data, key)
		delete(m.expiry, key)
		return "", nil
	}

	val, ok := m.data[key]
	if !ok {
		return "", nil
	}
	return val, nil
}

// Set sets a value with optional expiry.
func (m *MockRedisClient) Set(ctx context.Context, key, value string, expiry time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	if expiry > 0 {
		m.expiry[key] = time.Now().Add(expiry)
	}
	return nil
}

// Delete deletes keys.
func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
		delete(m.expiry, key)
	}
	return nil
}

// Exists checks if a key exists.
func (m *MockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check expiry
	if exp, ok := m.expiry[key]; ok && time.Now().After(exp) {
		return false, nil
	}

	_, ok := m.data[key]
	return ok, nil
}

// Incr increments a counter.
func (m *MockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[key]++
	return m.counters[key], nil
}

// Decr decrements a counter.
func (m *MockRedisClient) Decr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[key]--
	return m.counters[key], nil
}

// SAdd adds members to a set.
func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sets[key] == nil {
		m.sets[key] = make(map[string]struct{})
	}
	for _, member := range members {
		m.sets[key][member] = struct{}{}
	}
	return nil
}

// SMembers returns all members of a set.
func (m *MockRedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	set, ok := m.sets[key]
	if !ok {
		return []string{}, nil
	}

	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}
	return members, nil
}

// SRem removes members from a set.
func (m *MockRedisClient) SRem(ctx context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if set, ok := m.sets[key]; ok {
		for _, member := range members {
			delete(set, member)
		}
	}
	return nil
}

// ZAdd adds a member to a sorted set with a score.
func (m *MockRedisClient) ZAdd(ctx context.Context, key string, score float64, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.zsets[key] == nil {
		m.zsets[key] = make(map[string]float64)
	}
	m.zsets[key][member] = score
	return nil
}

// ZRem removes members from a sorted set.
func (m *MockRedisClient) ZRem(ctx context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if zset, ok := m.zsets[key]; ok {
		for _, member := range members {
			delete(zset, member)
		}
	}
	return nil
}

// ZScore returns the score of a member in a sorted set.
func (m *MockRedisClient) ZScore(ctx context.Context, key, member string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if zset, ok := m.zsets[key]; ok {
		if score, ok := zset[member]; ok {
			return score, nil
		}
	}
	return 0, nil
}

// Clear clears all data (useful for test cleanup).
func (m *MockRedisClient) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]string)
	m.expiry = make(map[string]time.Time)
	m.sets = make(map[string]map[string]struct{})
	m.zsets = make(map[string]map[string]float64)
	m.counters = make(map[string]int64)
}

// Keys returns all keys matching a pattern (simplified - exact match only).
func (m *MockRedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys, nil
}
