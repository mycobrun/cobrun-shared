//go:build integration

// Package database provides database client utilities.
package database

import (
	"testing"
	"time"

	pkgtesting "github.com/mycobrun/cobrun-shared/testing"
	"github.com/redis/go-redis/v9"
)

func TestRedisClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := pkgtesting.TestContext(t)

	// Start Redis container
	container, err := pkgtesting.StartRedisContainer(ctx)
	if err != nil {
		t.Fatalf("failed to start Redis container: %v", err)
	}
	t.Cleanup(pkgtesting.CleanupContainer(ctx, container))

	// Create Redis client using the container
	// Parse the connection string to get host and port
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	config := RedisConfig{
		Host:       host,
		Port:       port.Int(),
		TLSEnabled: false, // Container doesn't use TLS
		PoolSize:   10,
	}

	client, err := NewRedisClient(ctx, config)
	if err != nil {
		t.Fatalf("failed to create Redis client: %v", err)
	}
	defer client.Close()

	t.Run("Ping", func(t *testing.T) {
		err := client.Ping(ctx)
		if err != nil {
			t.Errorf("Ping failed: %v", err)
		}
	})

	t.Run("Set_Get", func(t *testing.T) {
		key := "test:key:1"
		value := "test-value"

		err := client.Set(ctx, key, value, 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		got, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if got != value {
			t.Errorf("Get returned %q, want %q", got, value)
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Set_WithExpiration", func(t *testing.T) {
		key := "test:key:expiring"
		value := "expiring-value"

		err := client.Set(ctx, key, value, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Value should exist
		got, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get failed immediately after set: %v", err)
		}
		if got != value {
			t.Errorf("Get returned %q, want %q", got, value)
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Value should be gone
		_, err = client.Get(ctx, key)
		if err != ErrKeyNotFound {
			t.Errorf("expected ErrKeyNotFound after expiration, got %v", err)
		}
	})

	t.Run("SetNX", func(t *testing.T) {
		key := "test:key:setnx"
		value := "first-value"

		// First SetNX should succeed
		acquired, err := client.SetNX(ctx, key, value, 0)
		if err != nil {
			t.Fatalf("first SetNX failed: %v", err)
		}
		if !acquired {
			t.Error("first SetNX should have acquired")
		}

		// Second SetNX should fail (key exists)
		acquired, err = client.SetNX(ctx, key, "second-value", 0)
		if err != nil {
			t.Fatalf("second SetNX failed: %v", err)
		}
		if acquired {
			t.Error("second SetNX should not have acquired")
		}

		// Value should be the first one
		got, _ := client.Get(ctx, key)
		if got != value {
			t.Errorf("value should be first value %q, got %q", value, got)
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("JSON_Operations", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		key := "test:key:json"
		original := TestStruct{Name: "test", Value: 42}

		err := client.SetJSON(ctx, key, original, 0)
		if err != nil {
			t.Fatalf("SetJSON failed: %v", err)
		}

		var got TestStruct
		err = client.GetJSON(ctx, key, &got)
		if err != nil {
			t.Fatalf("GetJSON failed: %v", err)
		}

		if got.Name != original.Name || got.Value != original.Value {
			t.Errorf("GetJSON returned %+v, want %+v", got, original)
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Hash_Operations", func(t *testing.T) {
		key := "test:hash:1"

		err := client.HSet(ctx, key, "field1", "value1", "field2", "value2")
		if err != nil {
			t.Fatalf("HSet failed: %v", err)
		}

		// Get single field
		val, err := client.HGet(ctx, key, "field1")
		if err != nil {
			t.Fatalf("HGet failed: %v", err)
		}
		if val != "value1" {
			t.Errorf("HGet returned %q, want %q", val, "value1")
		}

		// Get all fields
		all, err := client.HGetAll(ctx, key)
		if err != nil {
			t.Fatalf("HGetAll failed: %v", err)
		}
		if len(all) != 2 {
			t.Errorf("HGetAll returned %d fields, want 2", len(all))
		}

		// Delete field
		err = client.HDel(ctx, key, "field1")
		if err != nil {
			t.Fatalf("HDel failed: %v", err)
		}

		// Verify deletion
		_, err = client.HGet(ctx, key, "field1")
		if err != ErrKeyNotFound {
			t.Errorf("expected ErrKeyNotFound after HDel, got %v", err)
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Geo_Operations", func(t *testing.T) {
		key := "test:geo:drivers"

		// Add drivers
		err := client.GeoAdd(ctx, key,
			&redis.GeoLocation{Name: "driver1", Longitude: -122.4194, Latitude: 37.7749},
			&redis.GeoLocation{Name: "driver2", Longitude: -122.4094, Latitude: 37.7849},
		)
		if err != nil {
			t.Fatalf("GeoAdd failed: %v", err)
		}

		// Find nearby drivers
		locations, err := client.GeoRadius(ctx, key, -122.4194, 37.7749, &redis.GeoRadiusQuery{
			Radius:   5,
			Unit:     "km",
			WithDist: true,
		})
		if err != nil {
			t.Fatalf("GeoRadius failed: %v", err)
		}

		if len(locations) != 2 {
			t.Errorf("GeoRadius returned %d locations, want 2", len(locations))
		}

		// Remove driver
		err = client.GeoRemove(ctx, key, "driver1")
		if err != nil {
			t.Fatalf("GeoRemove failed: %v", err)
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Sorted_Set_Operations", func(t *testing.T) {
		key := "test:zset:leaderboard"

		// Add members
		err := client.ZAdd(ctx, key,
			redis.Z{Score: 100, Member: "player1"},
			redis.Z{Score: 200, Member: "player2"},
			redis.Z{Score: 150, Member: "player3"},
		)
		if err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}

		// Get ranking
		rank, err := client.ZRank(ctx, key, "player1")
		if err != nil {
			t.Fatalf("ZRank failed: %v", err)
		}
		if rank != 0 { // player1 has lowest score
			t.Errorf("ZRank returned %d, want 0", rank)
		}

		// Get count
		count, err := client.ZCard(ctx, key)
		if err != nil {
			t.Fatalf("ZCard failed: %v", err)
		}
		if count != 3 {
			t.Errorf("ZCard returned %d, want 3", count)
		}

		// Get range
		members, err := client.ZRange(ctx, key, 0, -1)
		if err != nil {
			t.Fatalf("ZRange failed: %v", err)
		}
		if len(members) != 3 {
			t.Errorf("ZRange returned %d members, want 3", len(members))
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("List_Operations", func(t *testing.T) {
		key := "test:list:queue"

		// Push items
		err := client.LPush(ctx, key, "item1", "item2", "item3")
		if err != nil {
			t.Fatalf("LPush failed: %v", err)
		}

		// Check length
		length, err := client.LLen(ctx, key)
		if err != nil {
			t.Fatalf("LLen failed: %v", err)
		}
		if length != 3 {
			t.Errorf("LLen returned %d, want 3", length)
		}

		// Pop item
		val, err := client.RPop(ctx, key)
		if err != nil {
			t.Fatalf("RPop failed: %v", err)
		}
		if val != "item1" {
			t.Errorf("RPop returned %q, want %q", val, "item1")
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Set_Operations", func(t *testing.T) {
		key := "test:set:members"

		// Add members
		added, err := client.SAdd(ctx, key, "member1", "member2", "member3")
		if err != nil {
			t.Fatalf("SAdd failed: %v", err)
		}
		if added != 3 {
			t.Errorf("SAdd returned %d, want 3", added)
		}

		// Check membership
		isMember, err := client.SIsMember(ctx, key, "member1")
		if err != nil {
			t.Fatalf("SIsMember failed: %v", err)
		}
		if !isMember {
			t.Error("SIsMember should return true for existing member")
		}

		isMember, err = client.SIsMember(ctx, key, "nonexistent")
		if err != nil {
			t.Fatalf("SIsMember failed: %v", err)
		}
		if isMember {
			t.Error("SIsMember should return false for nonexistent member")
		}

		// Get cardinality
		card, err := client.SCard(ctx, key)
		if err != nil {
			t.Fatalf("SCard failed: %v", err)
		}
		if card != 3 {
			t.Errorf("SCard returned %d, want 3", card)
		}

		// Remove member
		removed, err := client.SRem(ctx, key, "member1")
		if err != nil {
			t.Fatalf("SRem failed: %v", err)
		}
		if removed != 1 {
			t.Errorf("SRem returned %d, want 1", removed)
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Increment_Operations", func(t *testing.T) {
		key := "test:counter"

		// Increment
		val, err := client.Incr(ctx, key)
		if err != nil {
			t.Fatalf("Incr failed: %v", err)
		}
		if val != 1 {
			t.Errorf("Incr returned %d, want 1", val)
		}

		// Increment by amount
		val, err = client.IncrBy(ctx, key, 5)
		if err != nil {
			t.Fatalf("IncrBy failed: %v", err)
		}
		if val != 6 {
			t.Errorf("IncrBy returned %d, want 6", val)
		}

		// Decrement
		val, err = client.Decr(ctx, key)
		if err != nil {
			t.Fatalf("Decr failed: %v", err)
		}
		if val != 5 {
			t.Errorf("Decr returned %d, want 5", val)
		}

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Distributed_Lock", func(t *testing.T) {
		lockKey := "test:lock:resource"

		// Acquire lock
		lock, err := client.AcquireLock(ctx, lockKey, 5*time.Second)
		if err != nil {
			t.Fatalf("AcquireLock failed: %v", err)
		}

		// Try to acquire same lock (should fail)
		_, err = client.AcquireLock(ctx, lockKey, 5*time.Second)
		if err != ErrLockNotAcquired {
			t.Errorf("expected ErrLockNotAcquired, got %v", err)
		}

		// Release lock
		err = lock.Release(ctx)
		if err != nil {
			t.Fatalf("Release failed: %v", err)
		}

		// Now should be able to acquire again
		lock2, err := client.AcquireLock(ctx, lockKey, 5*time.Second)
		if err != nil {
			t.Fatalf("second AcquireLock failed: %v", err)
		}
		lock2.Release(ctx)
	})

	t.Run("PubSub", func(t *testing.T) {
		channel := "test:channel"
		message := "hello"

		// Subscribe
		pubsub := client.Subscribe(ctx, channel)
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(ctx)
		if err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}

		// Publish in goroutine
		go func() {
			time.Sleep(50 * time.Millisecond)
			client.Publish(ctx, channel, message)
		}()

		// Receive message
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			t.Fatalf("ReceiveMessage failed: %v", err)
		}

		if msg.Payload != message {
			t.Errorf("received %q, want %q", msg.Payload, message)
		}
	})
}

func TestRedisClient_IntegrationRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := pkgtesting.TestContext(t)

	// Start Redis container
	container, err := pkgtesting.StartRedisContainer(ctx)
	if err != nil {
		t.Fatalf("failed to start Redis container: %v", err)
	}
	t.Cleanup(pkgtesting.CleanupContainer(ctx, container))

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")

	config := RedisConfig{
		Host:       host,
		Port:       port.Int(),
		TLSEnabled: false,
		PoolSize:   10,
	}

	client, err := NewRedisClient(ctx, config)
	if err != nil {
		t.Fatalf("failed to create Redis client: %v", err)
	}
	defer client.Close()

	t.Run("SetWithRetry", func(t *testing.T) {
		key := "test:retry:key"
		value := "retry-value"

		err := client.SetWithRetry(ctx, key, value, 0)
		if err != nil {
			t.Fatalf("SetWithRetry failed: %v", err)
		}

		got, err := client.GetWithRetry(ctx, key)
		if err != nil {
			t.Fatalf("GetWithRetry failed: %v", err)
		}

		if got != value {
			t.Errorf("GetWithRetry returned %q, want %q", got, value)
		}

		client.DeleteWithRetry(ctx, key)
	})

	t.Run("AcquireLockWithRetry", func(t *testing.T) {
		lockKey := "test:lock:retry"

		// First lock
		lock1, err := client.AcquireLockWithRetry(ctx, lockKey, 100*time.Millisecond, 1)
		if err != nil {
			t.Fatalf("first AcquireLockWithRetry failed: %v", err)
		}

		// Second attempt with retry should eventually succeed after lock expires
		go func() {
			time.Sleep(50 * time.Millisecond)
			lock1.Release(ctx)
		}()

		lock2, err := client.AcquireLockWithRetry(ctx, lockKey, 5*time.Second, 5)
		if err != nil {
			t.Fatalf("second AcquireLockWithRetry failed: %v", err)
		}
		lock2.Release(ctx)
	})
}
