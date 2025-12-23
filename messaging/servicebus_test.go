package messaging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceBusConfig(t *testing.T) {
	t.Run("valid config with connection string", func(t *testing.T) {
		config := ServiceBusConfig{
			Namespace:        "test-namespace",
			ConnectionString: "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=testkey",
		}

		assert.Equal(t, "test-namespace", config.Namespace)
		assert.NotEmpty(t, config.ConnectionString)
	})

	t.Run("valid config with managed identity", func(t *testing.T) {
		config := ServiceBusConfig{
			Namespace: "test-namespace",
		}

		assert.Equal(t, "test-namespace", config.Namespace)
		assert.Empty(t, config.ConnectionString)
	})
}

func TestMessage(t *testing.T) {
	t.Run("create message with basic properties", func(t *testing.T) {
		msg := &Message{
			ID:          "msg-123",
			Body:        []byte("test message"),
			ContentType: "application/json",
		}

		assert.Equal(t, "msg-123", msg.ID)
		assert.Equal(t, []byte("test message"), msg.Body)
		assert.Equal(t, "application/json", msg.ContentType)
	})

	t.Run("create message with all properties", func(t *testing.T) {
		scheduledTime := time.Now().Add(1 * time.Hour)
		msg := &Message{
			ID:                   "msg-456",
			Body:                 []byte("test"),
			ContentType:          "application/json",
			CorrelationID:        "corr-123",
			Subject:              "test.subject",
			SessionID:            "session-1",
			ScheduledEnqueueTime: &scheduledTime,
			Properties: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		}

		assert.Equal(t, "msg-456", msg.ID)
		assert.Equal(t, "corr-123", msg.CorrelationID)
		assert.Equal(t, "test.subject", msg.Subject)
		assert.Equal(t, "session-1", msg.SessionID)
		assert.NotNil(t, msg.ScheduledEnqueueTime)
		assert.Equal(t, 2, len(msg.Properties))
	})

	t.Run("message with properties map", func(t *testing.T) {
		msg := &Message{
			ID:   "msg-789",
			Body: []byte("test"),
			Properties: map[string]interface{}{
				"source":      "system-a",
				"priority":    1,
				"retry_count": 3,
			},
		}

		assert.Equal(t, "system-a", msg.Properties["source"])
		assert.Equal(t, 1, msg.Properties["priority"])
		assert.Equal(t, 3, msg.Properties["retry_count"])
	})
}

func TestMessageOptions(t *testing.T) {
	t.Run("WithCorrelationID", func(t *testing.T) {
		msg := &Message{}
		opt := WithCorrelationID("corr-123")
		opt(msg)

		assert.Equal(t, "corr-123", msg.CorrelationID)
	})

	t.Run("WithSubject", func(t *testing.T) {
		msg := &Message{}
		opt := WithSubject("test.subject")
		opt(msg)

		assert.Equal(t, "test.subject", msg.Subject)
	})

	t.Run("WithSessionID", func(t *testing.T) {
		msg := &Message{}
		opt := WithSessionID("session-123")
		opt(msg)

		assert.Equal(t, "session-123", msg.SessionID)
	})

	t.Run("WithProperty", func(t *testing.T) {
		msg := &Message{}
		opt := WithProperty("key", "value")
		opt(msg)

		assert.NotNil(t, msg.Properties)
		assert.Equal(t, "value", msg.Properties["key"])
	})

	t.Run("WithProperty multiple properties", func(t *testing.T) {
		msg := &Message{}
		opt1 := WithProperty("key1", "value1")
		opt2 := WithProperty("key2", 123)
		opt3 := WithProperty("key3", true)

		opt1(msg)
		opt2(msg)
		opt3(msg)

		assert.Equal(t, 3, len(msg.Properties))
		assert.Equal(t, "value1", msg.Properties["key1"])
		assert.Equal(t, 123, msg.Properties["key2"])
		assert.Equal(t, true, msg.Properties["key3"])
	})

	t.Run("WithScheduledTime", func(t *testing.T) {
		msg := &Message{}
		scheduledTime := time.Now().Add(1 * time.Hour)
		opt := WithScheduledTime(scheduledTime)
		opt(msg)

		assert.NotNil(t, msg.ScheduledEnqueueTime)
		assert.Equal(t, scheduledTime, *msg.ScheduledEnqueueTime)
	})

	t.Run("multiple options combined", func(t *testing.T) {
		msg := &Message{
			ID:   "msg-1",
			Body: []byte("test"),
		}

		opts := []MessageOption{
			WithCorrelationID("corr-1"),
			WithSubject("test.subject"),
			WithProperty("source", "test"),
			WithSessionID("session-1"),
		}

		for _, opt := range opts {
			opt(msg)
		}

		assert.Equal(t, "corr-1", msg.CorrelationID)
		assert.Equal(t, "test.subject", msg.Subject)
		assert.Equal(t, "session-1", msg.SessionID)
		assert.Equal(t, "test", msg.Properties["source"])
	})
}

func TestReceivedMessage(t *testing.T) {
	t.Run("create received message", func(t *testing.T) {
		now := time.Now()
		msg := &ReceivedMessage{
			Message: Message{
				ID:            "msg-123",
				Body:          []byte("test message"),
				ContentType:   "application/json",
				CorrelationID: "corr-123",
			},
			EnqueuedTime:   now,
			DeliveryCount:  1,
			SequenceNumber: 100,
		}

		assert.Equal(t, "msg-123", msg.ID)
		assert.Equal(t, []byte("test message"), msg.Body)
		assert.Equal(t, "application/json", msg.ContentType)
		assert.Equal(t, "corr-123", msg.CorrelationID)
		assert.Equal(t, now, msg.EnqueuedTime)
		assert.Equal(t, uint32(1), msg.DeliveryCount)
		assert.Equal(t, int64(100), msg.SequenceNumber)
	})

	t.Run("received message with properties", func(t *testing.T) {
		msg := &ReceivedMessage{
			Message: Message{
				Body: []byte("test"),
				Properties: map[string]interface{}{
					"source":   "system-a",
					"priority": 1,
				},
			},
			DeliveryCount: 2,
		}

		assert.Equal(t, "system-a", msg.Properties["source"])
		assert.Equal(t, 1, msg.Properties["priority"])
		assert.Equal(t, uint32(2), msg.DeliveryCount)
	})
}

func TestReceivedMessageUnmarshalJSON(t *testing.T) {
	t.Run("unmarshal valid JSON", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "test",
			"value": 123,
		}
		body, err := json.Marshal(data)
		require.NoError(t, err)

		msg := &ReceivedMessage{
			Message: Message{
				Body: body,
			},
		}

		var result map[string]interface{}
		err = msg.UnmarshalJSON(&result)
		require.NoError(t, err)

		assert.Equal(t, "test", result["name"])
		assert.Equal(t, float64(123), result["value"])
	})

	t.Run("unmarshal struct", func(t *testing.T) {
		type TestData struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}

		original := TestData{
			Name:  "test",
			Count: 42,
		}
		body, err := json.Marshal(original)
		require.NoError(t, err)

		msg := &ReceivedMessage{
			Message: Message{
				Body: body,
			},
		}

		var result TestData
		err = msg.UnmarshalJSON(&result)
		require.NoError(t, err)

		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 42, result.Count)
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		msg := &ReceivedMessage{
			Message: Message{
				Body: []byte("invalid json"),
			},
		}

		var result map[string]interface{}
		err := msg.UnmarshalJSON(&result)
		assert.Error(t, err)
	})

	t.Run("unmarshal complex nested structure", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type Person struct {
			Name    string  `json:"name"`
			Age     int     `json:"age"`
			Address Address `json:"address"`
		}

		original := Person{
			Name: "John Doe",
			Age:  30,
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
		}

		body, err := json.Marshal(original)
		require.NoError(t, err)

		msg := &ReceivedMessage{
			Message: Message{
				Body: body,
			},
		}

		var result Person
		err = msg.UnmarshalJSON(&result)
		require.NoError(t, err)

		assert.Equal(t, "John Doe", result.Name)
		assert.Equal(t, 30, result.Age)
		assert.Equal(t, "123 Main St", result.Address.Street)
		assert.Equal(t, "New York", result.Address.City)
	})
}

func TestMessageHandler(t *testing.T) {
	t.Run("create and execute handler", func(t *testing.T) {
		var receivedMsg *ReceivedMessage
		handler := func(ctx context.Context, msg *ReceivedMessage) error {
			receivedMsg = msg
			return nil
		}

		testMsg := &ReceivedMessage{
			Message: Message{
				ID:   "msg-123",
				Body: []byte("test"),
			},
		}

		err := handler(context.Background(), testMsg)
		assert.NoError(t, err)
		assert.Equal(t, testMsg, receivedMsg)
	})

	t.Run("handler with error", func(t *testing.T) {
		handler := func(ctx context.Context, msg *ReceivedMessage) error {
			return assert.AnError
		}

		err := handler(context.Background(), &ReceivedMessage{})
		assert.Error(t, err)
	})

	t.Run("handler with context cancellation", func(t *testing.T) {
		handler := func(ctx context.Context, msg *ReceivedMessage) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := handler(ctx, &ReceivedMessage{})
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestSimpleMessageHandler(t *testing.T) {
	t.Run("create and execute simple handler", func(t *testing.T) {
		var receivedMsg *Message
		handler := func(msg *Message) error {
			receivedMsg = msg
			return nil
		}

		testMsg := &Message{
			ID:   "msg-456",
			Body: []byte("simple test"),
		}

		err := handler(testMsg)
		assert.NoError(t, err)
		assert.Equal(t, testMsg, receivedMsg)
	})

	t.Run("simple handler with error", func(t *testing.T) {
		handler := func(msg *Message) error {
			return assert.AnError
		}

		err := handler(&Message{})
		assert.Error(t, err)
	})
}

func TestMessageSerialization(t *testing.T) {
	t.Run("serialize message", func(t *testing.T) {
		msg := &Message{
			ID:            "msg-123",
			Body:          []byte("test message"),
			ContentType:   "application/json",
			CorrelationID: "corr-123",
			Subject:       "test.subject",
			Properties: map[string]interface{}{
				"key": "value",
			},
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded Message
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msg.ID, decoded.ID)
		assert.Equal(t, msg.ContentType, decoded.ContentType)
		assert.Equal(t, msg.CorrelationID, decoded.CorrelationID)
		assert.Equal(t, msg.Subject, decoded.Subject)
	})

	t.Run("serialize received message", func(t *testing.T) {
		now := time.Now()
		msg := &ReceivedMessage{
			Message: Message{
				ID:   "msg-456",
				Body: []byte("test"),
			},
			EnqueuedTime:   now,
			DeliveryCount:  2,
			SequenceNumber: 200,
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}

func TestMessageWithComplexData(t *testing.T) {
	t.Run("message with nested JSON payload", func(t *testing.T) {
		type OrderItem struct {
			ProductID string  `json:"product_id"`
			Quantity  int     `json:"quantity"`
			Price     float64 `json:"price"`
		}

		type Order struct {
			OrderID    string      `json:"order_id"`
			CustomerID string      `json:"customer_id"`
			Items      []OrderItem `json:"items"`
			Total      float64     `json:"total"`
		}

		order := Order{
			OrderID:    "order-123",
			CustomerID: "customer-456",
			Items: []OrderItem{
				{ProductID: "prod-1", Quantity: 2, Price: 10.50},
				{ProductID: "prod-2", Quantity: 1, Price: 25.00},
			},
			Total: 46.00,
		}

		body, err := json.Marshal(order)
		require.NoError(t, err)

		msg := &Message{
			ID:          "msg-789",
			Body:        body,
			ContentType: "application/json",
		}

		// Verify we can unmarshal back
		var decoded Order
		err = json.Unmarshal(msg.Body, &decoded)
		require.NoError(t, err)

		assert.Equal(t, order.OrderID, decoded.OrderID)
		assert.Equal(t, 2, len(decoded.Items))
		assert.Equal(t, 46.00, decoded.Total)
	})
}

func TestMessagePropertyTypes(t *testing.T) {
	t.Run("properties with various types", func(t *testing.T) {
		msg := &Message{
			ID:   "msg-1",
			Body: []byte("test"),
			Properties: map[string]interface{}{
				"string_prop": "value",
				"int_prop":    42,
				"float_prop":  3.14,
				"bool_prop":   true,
				"array_prop":  []string{"a", "b", "c"},
				"map_prop": map[string]string{
					"nested": "value",
				},
			},
		}

		assert.Equal(t, "value", msg.Properties["string_prop"])
		assert.Equal(t, 42, msg.Properties["int_prop"])
		assert.Equal(t, 3.14, msg.Properties["float_prop"])
		assert.Equal(t, true, msg.Properties["bool_prop"])
		assert.NotNil(t, msg.Properties["array_prop"])
		assert.NotNil(t, msg.Properties["map_prop"])
	})
}

func TestScheduledMessage(t *testing.T) {
	t.Run("message with scheduled time", func(t *testing.T) {
		scheduledTime := time.Now().Add(2 * time.Hour)
		msg := &Message{
			ID:                   "msg-scheduled",
			Body:                 []byte("scheduled message"),
			ScheduledEnqueueTime: &scheduledTime,
		}

		assert.NotNil(t, msg.ScheduledEnqueueTime)
		assert.True(t, msg.ScheduledEnqueueTime.After(time.Now()))
	})

	t.Run("message without scheduled time", func(t *testing.T) {
		msg := &Message{
			ID:   "msg-immediate",
			Body: []byte("immediate message"),
		}

		assert.Nil(t, msg.ScheduledEnqueueTime)
	})
}

func TestSessionMessage(t *testing.T) {
	t.Run("message with session ID", func(t *testing.T) {
		msg := &Message{
			ID:        "msg-session",
			Body:      []byte("session message"),
			SessionID: "session-123",
		}

		assert.Equal(t, "session-123", msg.SessionID)
	})

	t.Run("related messages in same session", func(t *testing.T) {
		sessionID := "session-456"

		msg1 := &Message{
			ID:        "msg-1",
			Body:      []byte("first"),
			SessionID: sessionID,
		}

		msg2 := &Message{
			ID:        "msg-2",
			Body:      []byte("second"),
			SessionID: sessionID,
		}

		assert.Equal(t, msg1.SessionID, msg2.SessionID)
	})
}

func TestMessageWithSubject(t *testing.T) {
	t.Run("message with subject filter", func(t *testing.T) {
		msg := &Message{
			ID:      "msg-1",
			Body:    []byte("test"),
			Subject: "orders.created",
		}

		assert.Equal(t, "orders.created", msg.Subject)
	})

	t.Run("messages with different subjects", func(t *testing.T) {
		msg1 := &Message{
			ID:      "msg-1",
			Body:    []byte("test"),
			Subject: "orders.created",
		}

		msg2 := &Message{
			ID:      "msg-2",
			Body:    []byte("test"),
			Subject: "orders.updated",
		}

		msg3 := &Message{
			ID:      "msg-3",
			Body:    []byte("test"),
			Subject: "orders.deleted",
		}

		assert.NotEqual(t, msg1.Subject, msg2.Subject)
		assert.NotEqual(t, msg2.Subject, msg3.Subject)
	})
}

func TestCorrelationScenarios(t *testing.T) {
	t.Run("request-response correlation", func(t *testing.T) {
		correlationID := "corr-request-123"

		requestMsg := &Message{
			ID:            "msg-request",
			Body:          []byte("request"),
			CorrelationID: correlationID,
		}

		responseMsg := &Message{
			ID:            "msg-response",
			Body:          []byte("response"),
			CorrelationID: correlationID,
		}

		assert.Equal(t, requestMsg.CorrelationID, responseMsg.CorrelationID)
	})
}

func TestReceivedMessageMetadata(t *testing.T) {
	t.Run("received message with metadata", func(t *testing.T) {
		now := time.Now()
		msg := &ReceivedMessage{
			Message: Message{
				ID:   "msg-123",
				Body: []byte("test"),
			},
			EnqueuedTime:   now,
			DeliveryCount:  3,
			SequenceNumber: 12345,
		}

		assert.Equal(t, now, msg.EnqueuedTime)
		assert.Equal(t, uint32(3), msg.DeliveryCount)
		assert.Equal(t, int64(12345), msg.SequenceNumber)
	})

	t.Run("check delivery count for retry logic", func(t *testing.T) {
		msg := &ReceivedMessage{
			Message: Message{
				ID:   "msg-retry",
				Body: []byte("test"),
			},
			DeliveryCount: 5,
		}

		maxRetries := uint32(3)
		shouldDeadLetter := msg.DeliveryCount > maxRetries
		assert.True(t, shouldDeadLetter)
	})
}
