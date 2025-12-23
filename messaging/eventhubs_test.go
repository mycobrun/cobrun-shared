package messaging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventHubsConfig(t *testing.T) {
	t.Run("valid config with connection string", func(t *testing.T) {
		config := EventHubsConfig{
			Namespace:        "test-namespace",
			EventHubName:     "test-hub",
			ConnectionString: "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test",
			ConsumerGroup:    "$Default",
		}

		assert.Equal(t, "test-namespace", config.Namespace)
		assert.Equal(t, "test-hub", config.EventHubName)
		assert.NotEmpty(t, config.ConnectionString)
		assert.Equal(t, "$Default", config.ConsumerGroup)
	})

	t.Run("valid config with managed identity", func(t *testing.T) {
		config := EventHubsConfig{
			Namespace:     "test-namespace",
			EventHubName:  "test-hub",
			ConsumerGroup: "custom-group",
		}

		assert.Equal(t, "test-namespace", config.Namespace)
		assert.Equal(t, "test-hub", config.EventHubName)
		assert.Empty(t, config.ConnectionString)
		assert.Equal(t, "custom-group", config.ConsumerGroup)
	})
}

func TestEvent(t *testing.T) {
	t.Run("create event with basic properties", func(t *testing.T) {
		body := []byte("test message")
		event := &Event{
			Body:        body,
			ContentType: "application/json",
		}

		assert.Equal(t, body, event.Body)
		assert.Equal(t, "application/json", event.ContentType)
	})

	t.Run("create event with properties", func(t *testing.T) {
		event := &Event{
			Body:         []byte("test"),
			ContentType:  "application/json",
			PartitionKey: "key1",
			Properties: map[string]string{
				"source": "test",
				"type":   "event",
			},
		}

		assert.Equal(t, "key1", event.PartitionKey)
		assert.Equal(t, "test", event.Properties["source"])
		assert.Equal(t, "event", event.Properties["type"])
	})
}

func TestEventOptions(t *testing.T) {
	t.Run("WithEventProperty", func(t *testing.T) {
		event := &Event{}
		opt := WithEventProperty("key", "value")
		opt(event)

		assert.NotNil(t, event.Properties)
		assert.Equal(t, "value", event.Properties["key"])
	})

	t.Run("WithEventProperty multiple properties", func(t *testing.T) {
		event := &Event{}
		opt1 := WithEventProperty("key1", "value1")
		opt2 := WithEventProperty("key2", "value2")
		opt1(event)
		opt2(event)

		assert.Equal(t, 2, len(event.Properties))
		assert.Equal(t, "value1", event.Properties["key1"])
		assert.Equal(t, "value2", event.Properties["key2"])
	})

	t.Run("WithPartitionKey", func(t *testing.T) {
		event := &Event{}
		opt := WithPartitionKey("partition-1")
		opt(event)

		assert.Equal(t, "partition-1", event.PartitionKey)
	})

	t.Run("multiple options", func(t *testing.T) {
		event := &Event{}
		opts := []EventOption{
			WithEventProperty("source", "test"),
			WithPartitionKey("key1"),
		}

		for _, opt := range opts {
			opt(event)
		}

		assert.Equal(t, "test", event.Properties["source"])
		assert.Equal(t, "key1", event.PartitionKey)
	})
}

func TestStartPosition(t *testing.T) {
	t.Run("StartPositionLatest", func(t *testing.T) {
		pos := StartPositionLatest()
		assert.NotNil(t, pos.Latest)
		assert.True(t, *pos.Latest)
	})

	t.Run("StartPositionEarliest", func(t *testing.T) {
		pos := StartPositionEarliest()
		assert.NotNil(t, pos.Earliest)
		assert.True(t, *pos.Earliest)
	})
}

func TestReceivedEvent(t *testing.T) {
	t.Run("create received event", func(t *testing.T) {
		now := time.Now()
		event := &ReceivedEvent{
			Body:           []byte("test message"),
			ContentType:    "application/json",
			EnqueuedTime:   now,
			SequenceNumber: 123,
			Offset:         456,
			PartitionKey:   "key1",
			PartitionID:    "0",
		}

		assert.Equal(t, []byte("test message"), event.Body)
		assert.Equal(t, "application/json", event.ContentType)
		assert.Equal(t, now, event.EnqueuedTime)
		assert.Equal(t, int64(123), event.SequenceNumber)
		assert.Equal(t, int64(456), event.Offset)
		assert.Equal(t, "key1", event.PartitionKey)
		assert.Equal(t, "0", event.PartitionID)
	})

	t.Run("received event with properties", func(t *testing.T) {
		event := &ReceivedEvent{
			Body: []byte("test"),
			Properties: map[string]string{
				"source": "test",
				"type":   "event",
			},
		}

		assert.Equal(t, "test", event.Properties["source"])
		assert.Equal(t, "event", event.Properties["type"])
	})
}

func TestReceivedEventUnmarshalJSON(t *testing.T) {
	t.Run("unmarshal valid JSON", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "test",
			"value": 123,
		}
		body, err := json.Marshal(data)
		require.NoError(t, err)

		event := &ReceivedEvent{
			Body: body,
		}

		var result map[string]interface{}
		err = event.UnmarshalJSON(&result)
		require.NoError(t, err)

		assert.Equal(t, "test", result["name"])
		assert.Equal(t, float64(123), result["value"]) // JSON numbers are float64
	})

	t.Run("unmarshal struct", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}

		original := TestStruct{
			Name:  "test",
			Count: 42,
		}
		body, err := json.Marshal(original)
		require.NoError(t, err)

		event := &ReceivedEvent{
			Body: body,
		}

		var result TestStruct
		err = event.UnmarshalJSON(&result)
		require.NoError(t, err)

		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 42, result.Count)
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		event := &ReceivedEvent{
			Body: []byte("invalid json"),
		}

		var result map[string]interface{}
		err := event.UnmarshalJSON(&result)
		assert.Error(t, err)
	})
}

func TestEventHubsConsumerConfig(t *testing.T) {
	t.Run("create consumer config", func(t *testing.T) {
		config := EventHubsConsumerConfig{
			EventHubsConfig: EventHubsConfig{
				Namespace:        "test-namespace",
				EventHubName:     "test-hub",
				ConnectionString: "test-connection",
				ConsumerGroup:    "$Default",
			},
			StorageConnectionString: "storage-connection",
			StorageContainerName:    "checkpoints",
		}

		assert.Equal(t, "test-namespace", config.Namespace)
		assert.Equal(t, "test-hub", config.EventHubName)
		assert.Equal(t, "storage-connection", config.StorageConnectionString)
		assert.Equal(t, "checkpoints", config.StorageContainerName)
	})
}

func TestToPtr(t *testing.T) {
	t.Run("toPtr bool", func(t *testing.T) {
		val := true
		ptr := toPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, true, *ptr)
	})

	t.Run("toPtr string", func(t *testing.T) {
		val := "test"
		ptr := toPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, "test", *ptr)
	})

	t.Run("toPtr int", func(t *testing.T) {
		val := 42
		ptr := toPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, 42, *ptr)
	})
}

func TestEventHandler(t *testing.T) {
	t.Run("create and execute event handler", func(t *testing.T) {
		var receivedEvent *ReceivedEvent
		handler := func(ctx context.Context, event *ReceivedEvent) error {
			receivedEvent = event
			return nil
		}

		testEvent := &ReceivedEvent{
			Body:        []byte("test"),
			PartitionID: "0",
		}

		err := handler(context.Background(), testEvent)
		assert.NoError(t, err)
		assert.Equal(t, testEvent, receivedEvent)
	})

	t.Run("handler with error", func(t *testing.T) {
		handler := func(ctx context.Context, event *ReceivedEvent) error {
			return assert.AnError
		}

		err := handler(context.Background(), &ReceivedEvent{})
		assert.Error(t, err)
	})

	t.Run("handler with context cancellation", func(t *testing.T) {
		handler := func(ctx context.Context, event *ReceivedEvent) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := handler(ctx, &ReceivedEvent{})
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestEventSerialization(t *testing.T) {
	t.Run("serialize event to JSON", func(t *testing.T) {
		event := &Event{
			Body:        []byte("test message"),
			ContentType: "application/json",
			Properties: map[string]string{
				"key": "value",
			},
		}

		// Serialize the event properties
		data, err := json.Marshal(event)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Deserialize back
		var decoded Event
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, event.ContentType, decoded.ContentType)
	})
}

func TestReceivedEventSerialization(t *testing.T) {
	t.Run("serialize received event", func(t *testing.T) {
		now := time.Now()
		event := &ReceivedEvent{
			Body:           []byte("test"),
			ContentType:    "text/plain",
			EnqueuedTime:   now,
			SequenceNumber: 100,
			Offset:         200,
			PartitionKey:   "key",
			PartitionID:    "1",
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded ReceivedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, event.ContentType, decoded.ContentType)
		assert.Equal(t, event.SequenceNumber, decoded.SequenceNumber)
	})
}

func TestEventWithComplexData(t *testing.T) {
	t.Run("event with nested JSON", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type User struct {
			Name    string  `json:"name"`
			Age     int     `json:"age"`
			Address Address `json:"address"`
		}

		user := User{
			Name: "John Doe",
			Age:  30,
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
		}

		body, err := json.Marshal(user)
		require.NoError(t, err)

		event := &Event{
			Body:        body,
			ContentType: "application/json",
		}

		assert.NotEmpty(t, event.Body)

		// Verify we can unmarshal it back
		var decoded User
		err = json.Unmarshal(event.Body, &decoded)
		require.NoError(t, err)
		assert.Equal(t, user.Name, decoded.Name)
		assert.Equal(t, user.Address.City, decoded.Address.City)
	})
}

func TestEventPropertiesManipulation(t *testing.T) {
	t.Run("add properties incrementally", func(t *testing.T) {
		event := &Event{
			Body:       []byte("test"),
			Properties: make(map[string]string),
		}

		event.Properties["key1"] = "value1"
		event.Properties["key2"] = "value2"

		assert.Equal(t, 2, len(event.Properties))
		assert.Equal(t, "value1", event.Properties["key1"])
		assert.Equal(t, "value2", event.Properties["key2"])
	})

	t.Run("override property", func(t *testing.T) {
		event := &Event{
			Body: []byte("test"),
			Properties: map[string]string{
				"key": "original",
			},
		}

		event.Properties["key"] = "updated"
		assert.Equal(t, "updated", event.Properties["key"])
	})
}

func TestReceivedEventProperties(t *testing.T) {
	t.Run("received event with multiple properties", func(t *testing.T) {
		event := &ReceivedEvent{
			Body:        []byte("test"),
			ContentType: "application/json",
			Properties: map[string]string{
				"source":      "system-a",
				"correlationId": "12345",
				"version":     "1.0",
			},
		}

		assert.Equal(t, 3, len(event.Properties))
		assert.Equal(t, "system-a", event.Properties["source"])
		assert.Equal(t, "12345", event.Properties["correlationId"])
		assert.Equal(t, "1.0", event.Properties["version"])
	})
}
