// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MockEvent represents a captured event.
type MockEvent struct {
	Topic     string
	Payload   interface{}
	Timestamp time.Time
}

// MockEventHubsClient is a mock implementation of Event Hubs for testing.
type MockEventHubsClient struct {
	mu         sync.RWMutex
	events     []MockEvent
	handlers   map[string][]func(context.Context, []byte) error
	shouldFail bool
	failError  error
}

// NewMockEventHubsClient creates a new mock Event Hubs client.
func NewMockEventHubsClient() *MockEventHubsClient {
	return &MockEventHubsClient{
		events:   make([]MockEvent, 0),
		handlers: make(map[string][]func(context.Context, []byte) error),
	}
}

// Publish publishes an event to a topic.
func (m *MockEventHubsClient) Publish(ctx context.Context, topic string, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return m.failError
	}

	event := MockEvent{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	m.events = append(m.events, event)

	// Invoke handlers
	if handlers, ok := m.handlers[topic]; ok {
		data, _ := json.Marshal(payload)
		for _, handler := range handlers {
			go handler(ctx, data)
		}
	}

	return nil
}

// Subscribe subscribes to a topic.
func (m *MockEventHubsClient) Subscribe(ctx context.Context, topic string, handler func(context.Context, []byte) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.handlers[topic] == nil {
		m.handlers[topic] = make([]func(context.Context, []byte) error, 0)
	}
	m.handlers[topic] = append(m.handlers[topic], handler)

	return nil
}

// GetEvents returns all captured events.
func (m *MockEventHubsClient) GetEvents() []MockEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]MockEvent{}, m.events...)
}

// GetEventsByTopic returns events for a specific topic.
func (m *MockEventHubsClient) GetEventsByTopic(topic string) []MockEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filtered := make([]MockEvent, 0)
	for _, event := range m.events {
		if event.Topic == topic {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// GetEventCount returns the number of events.
func (m *MockEventHubsClient) GetEventCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.events)
}

// GetEventCountByTopic returns the number of events for a topic.
func (m *MockEventHubsClient) GetEventCountByTopic(topic string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, event := range m.events {
		if event.Topic == topic {
			count++
		}
	}
	return count
}

// Clear clears all events.
func (m *MockEventHubsClient) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = make([]MockEvent, 0)
}

// SetShouldFail sets whether the client should fail on publish.
func (m *MockEventHubsClient) SetShouldFail(shouldFail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shouldFail = shouldFail
	m.failError = err
}

// WaitForEvent waits for an event on a topic with a timeout.
func (m *MockEventHubsClient) WaitForEvent(topic string, timeout time.Duration) (*MockEvent, bool) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		m.mu.RLock()
		for i := len(m.events) - 1; i >= 0; i-- {
			if m.events[i].Topic == topic {
				event := m.events[i]
				m.mu.RUnlock()
				return &event, true
			}
		}
		m.mu.RUnlock()
		time.Sleep(10 * time.Millisecond)
	}

	return nil, false
}

// AssertEventPublished asserts that an event was published to a topic.
func (m *MockEventHubsClient) AssertEventPublished(topic string) bool {
	return m.GetEventCountByTopic(topic) > 0
}

// AssertNoEvents asserts that no events were published.
func (m *MockEventHubsClient) AssertNoEvents() bool {
	return m.GetEventCount() == 0
}
