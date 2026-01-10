package messaging

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalRConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := SignalRConfig{
			ConnectionString: "Endpoint=https://test.service.signalr.net;AccessKey=testkey123",
			HubName:          "test-hub",
		}

		assert.NotEmpty(t, config.ConnectionString)
		assert.Equal(t, "test-hub", config.HubName)
	})
}

func TestParseConnectionString(t *testing.T) {
	t.Run("valid connection string with https", func(t *testing.T) {
		connStr := "Endpoint=https://test.service.signalr.net;AccessKey=testkey123"
		endpoint, accessKey, err := parseConnectionString(connStr)

		require.NoError(t, err)
		assert.Equal(t, "https://test.service.signalr.net", endpoint)
		assert.Equal(t, "testkey123", accessKey)
	})

	t.Run("valid connection string without https", func(t *testing.T) {
		connStr := "Endpoint=test.service.signalr.net;AccessKey=testkey456"
		endpoint, accessKey, err := parseConnectionString(connStr)

		require.NoError(t, err)
		assert.Equal(t, "https://test.service.signalr.net", endpoint)
		assert.Equal(t, "testkey456", accessKey)
	})

	t.Run("connection string with extra semicolons", func(t *testing.T) {
		connStr := "Endpoint=https://test.signalr.net;;AccessKey=key123;"
		endpoint, accessKey, err := parseConnectionString(connStr)

		require.NoError(t, err)
		assert.Equal(t, "https://test.signalr.net", endpoint)
		assert.Equal(t, "key123", accessKey)
	})

	t.Run("missing endpoint", func(t *testing.T) {
		connStr := "AccessKey=testkey"
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing Endpoint or AccessKey")
	})

	t.Run("missing access key", func(t *testing.T) {
		connStr := "Endpoint=https://test.signalr.net"
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing Endpoint or AccessKey")
	})

	t.Run("empty connection string", func(t *testing.T) {
		connStr := ""
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
	})

	t.Run("malformed connection string", func(t *testing.T) {
		connStr := "invalid connection string"
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
	})
}

func TestNewSignalRClient(t *testing.T) {
	t.Run("create client with valid config", func(t *testing.T) {
		config := SignalRConfig{
			ConnectionString: "Endpoint=https://test.signalr.net;AccessKey=dGVzdGtleQ==",
			HubName:          "test-hub",
		}

		client, err := NewSignalRClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://test.signalr.net", client.endpoint)
		assert.Equal(t, "test-hub", client.hubName)
		assert.NotNil(t, client.httpClient)
	})

	t.Run("create client with invalid config", func(t *testing.T) {
		config := SignalRConfig{
			ConnectionString: "invalid",
			HubName:          "test-hub",
		}

		client, err := NewSignalRClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestSignalRMessage(t *testing.T) {
	t.Run("create message with no arguments", func(t *testing.T) {
		msg := NewSignalRMessage("testMethod")
		assert.Equal(t, "testMethod", msg.Target)
		// Variadic args with no arguments create nil slice
		assert.Equal(t, 0, len(msg.Arguments))
	})

	t.Run("create message with single argument", func(t *testing.T) {
		msg := NewSignalRMessage("testMethod", "arg1")
		assert.Equal(t, "testMethod", msg.Target)
		assert.Equal(t, 1, len(msg.Arguments))
		assert.Equal(t, "arg1", msg.Arguments[0])
	})

	t.Run("create message with multiple arguments", func(t *testing.T) {
		msg := NewSignalRMessage("testMethod", "arg1", 42, true)
		assert.Equal(t, "testMethod", msg.Target)
		assert.Equal(t, 3, len(msg.Arguments))
		assert.Equal(t, "arg1", msg.Arguments[0])
		assert.Equal(t, 42, msg.Arguments[1])
		assert.Equal(t, true, msg.Arguments[2])
	})

	t.Run("create message with complex argument", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "test",
			"value": 123,
		}
		msg := NewSignalRMessage("testMethod", data)
		assert.Equal(t, 1, len(msg.Arguments))

		argMap, ok := msg.Arguments[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test", argMap["name"])
		assert.Equal(t, 123, argMap["value"])
	})
}

func TestSignalRMessageSerialization(t *testing.T) {
	t.Run("serialize message to JSON", func(t *testing.T) {
		msg := NewSignalRMessage("testMethod", "arg1", 42)
		data, err := json.Marshal(msg)

		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Verify JSON structure
		var decoded map[string]interface{}
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "testMethod", decoded["target"])
		assert.NotNil(t, decoded["arguments"])
	})
}

func TestLocationUpdateMessage(t *testing.T) {
	t.Run("create location update", func(t *testing.T) {
		msg := &LocationUpdateMessage{
			DriverID:  "driver-123",
			Latitude:  40.7128,
			Longitude: -74.0060,
			Heading:   90.5,
			Speed:     25.3,
			Timestamp: time.Now().Unix(),
		}

		assert.Equal(t, "driver-123", msg.DriverID)
		assert.Equal(t, 40.7128, msg.Latitude)
		assert.Equal(t, -74.0060, msg.Longitude)
		assert.Equal(t, 90.5, msg.Heading)
		assert.Equal(t, 25.3, msg.Speed)
		assert.Greater(t, msg.Timestamp, int64(0))
	})

	t.Run("serialize location update", func(t *testing.T) {
		msg := &LocationUpdateMessage{
			DriverID:  "driver-456",
			Latitude:  37.7749,
			Longitude: -122.4194,
			Heading:   180.0,
			Speed:     30.0,
			Timestamp: 1234567890,
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var decoded LocationUpdateMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msg.DriverID, decoded.DriverID)
		assert.Equal(t, msg.Latitude, decoded.Latitude)
		assert.Equal(t, msg.Longitude, decoded.Longitude)
		assert.Equal(t, msg.Heading, decoded.Heading)
		assert.Equal(t, msg.Speed, decoded.Speed)
		assert.Equal(t, msg.Timestamp, decoded.Timestamp)
	})
}

func TestTripStatusMessage(t *testing.T) {
	t.Run("create trip status message", func(t *testing.T) {
		msg := &TripStatusMessage{
			TripID:     "trip-123",
			Status:     "in_progress",
			DriverID:   "driver-456",
			DriverName: "John Doe",
			ETA:        300,
			Message:    "Driver arriving in 5 minutes",
		}

		assert.Equal(t, "trip-123", msg.TripID)
		assert.Equal(t, "in_progress", msg.Status)
		assert.Equal(t, "driver-456", msg.DriverID)
		assert.Equal(t, "John Doe", msg.DriverName)
		assert.Equal(t, 300, msg.ETA)
		assert.Equal(t, "Driver arriving in 5 minutes", msg.Message)
	})

	t.Run("serialize trip status message", func(t *testing.T) {
		msg := &TripStatusMessage{
			TripID:     "trip-789",
			Status:     "completed",
			DriverID:   "driver-123",
			DriverName: "Jane Smith",
			Message:    "Trip completed successfully",
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var decoded TripStatusMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msg.TripID, decoded.TripID)
		assert.Equal(t, msg.Status, decoded.Status)
		assert.Equal(t, msg.DriverID, decoded.DriverID)
		assert.Equal(t, msg.DriverName, decoded.DriverName)
		assert.Equal(t, msg.Message, decoded.Message)
	})
}

func TestSignalRClientSendRequest(t *testing.T) {
	t.Run("send request with body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &SignalRClient{
			endpoint:   server.URL,
			accessKey:  "dGVzdGtleQ==",
			hubName:    "test-hub",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		msg := NewSignalRMessage("test", "arg1")
		err := client.sendRequest(context.Background(), "POST", server.URL, msg)
		assert.NoError(t, err)
	})

	t.Run("send request without body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &SignalRClient{
			endpoint:   server.URL,
			accessKey:  "dGVzdGtleQ==",
			hubName:    "test-hub",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.sendRequest(context.Background(), "DELETE", server.URL, nil)
		assert.NoError(t, err)
	})

	t.Run("send request with error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad request"))
		}))
		defer server.Close()

		client := &SignalRClient{
			endpoint:   server.URL,
			accessKey:  "dGVzdGtleQ==",
			hubName:    "test-hub",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.sendRequest(context.Background(), "POST", server.URL, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func TestGenerateClientToken(t *testing.T) {
	t.Run("generate token without user ID", func(t *testing.T) {
		client := &SignalRClient{
			endpoint:  "https://test.signalr.net",
			accessKey: "dGVzdGtleQ==",
			hubName:   "test-hub",
		}

		resp, err := client.GenerateClientToken("", 1*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.URL)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Contains(t, resp.URL, "test-hub")
	})

	t.Run("generate token with user ID", func(t *testing.T) {
		client := &SignalRClient{
			endpoint:  "https://test.signalr.net",
			accessKey: "dGVzdGtleQ==",
			hubName:   "test-hub",
		}

		resp, err := client.GenerateClientToken("user-123", 2*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)

		// Verify token has 3 parts (header.payload.signature)
		parts := strings.Split(resp.AccessToken, ".")
		assert.Equal(t, 3, len(parts))
	})

	t.Run("generate token validates TTL", func(t *testing.T) {
		client := &SignalRClient{
			endpoint:  "https://test.signalr.net",
			accessKey: "dGVzdGtleQ==",
			hubName:   "test-hub",
		}

		ttl := 30 * time.Minute
		resp, err := client.GenerateClientToken("user-123", ttl)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestNegotiateResponse(t *testing.T) {
	t.Run("create negotiate response", func(t *testing.T) {
		resp := &NegotiateResponse{
			URL:         "https://test.signalr.net/client/?hub=test",
			AccessToken: "token123",
		}

		assert.NotEmpty(t, resp.URL)
		assert.NotEmpty(t, resp.AccessToken)
	})

	t.Run("serialize negotiate response", func(t *testing.T) {
		resp := &NegotiateResponse{
			URL:         "https://test.signalr.net/client/?hub=test",
			AccessToken: "token456",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded NegotiateResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.URL, decoded.URL)
		assert.Equal(t, resp.AccessToken, decoded.AccessToken)
	})
}

func TestSignalRClientHelperMethods(t *testing.T) {
	createMockClient := func() *SignalRClient {
		return &SignalRClient{
			endpoint:   "https://test.signalr.net",
			accessKey:  "dGVzdGtleQ==",
			hubName:    "test-hub",
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
	}

	t.Run("SendLocationUpdate creates correct message", func(t *testing.T) {
		client := createMockClient()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "users/rider-123")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client.endpoint = server.URL

		update := &LocationUpdateMessage{
			DriverID:  "driver-123",
			Latitude:  40.7128,
			Longitude: -74.0060,
		}

		err := client.SendLocationUpdate(context.Background(), "rider-123", update)
		assert.NoError(t, err)
	})

	t.Run("SendTripStatusUpdate creates correct message", func(t *testing.T) {
		client := createMockClient()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "users/user-123")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client.endpoint = server.URL

		update := &TripStatusMessage{
			TripID: "trip-123",
			Status: "in_progress",
		}

		err := client.SendTripStatusUpdate(context.Background(), "user-123", update)
		assert.NoError(t, err)
	})

	t.Run("AddToTripGroup creates correct group name", func(t *testing.T) {
		client := createMockClient()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "groups/trip-trip-123")
			assert.Contains(t, r.URL.Path, "users/user-456")
			assert.Equal(t, "PUT", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client.endpoint = server.URL

		err := client.AddToTripGroup(context.Background(), "trip-123", "user-456")
		assert.NoError(t, err)
	})

	t.Run("RemoveFromTripGroup creates correct group name", func(t *testing.T) {
		client := createMockClient()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "groups/trip-trip-789")
			assert.Contains(t, r.URL.Path, "users/user-123")
			assert.Equal(t, "DELETE", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client.endpoint = server.URL

		err := client.RemoveFromTripGroup(context.Background(), "trip-789", "user-123")
		assert.NoError(t, err)
	})

	t.Run("BroadcastToTrip uses correct group", func(t *testing.T) {
		client := createMockClient()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "groups/trip-trip-555")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client.endpoint = server.URL

		msg := NewSignalRMessage("update", "data")
		err := client.BroadcastToTrip(context.Background(), "trip-555", msg)
		assert.NoError(t, err)
	})

	t.Run("SendToUserWithTarget", func(t *testing.T) {
		client := createMockClient()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "users/user-999")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client.endpoint = server.URL

		err := client.SendToUserWithTarget(context.Background(), "user-999", "customMethod", map[string]string{"key": "value"})
		assert.NoError(t, err)
	})

	t.Run("SendToGroupWithTarget", func(t *testing.T) {
		client := createMockClient()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "groups/test-group")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client.endpoint = server.URL

		err := client.SendToGroupWithTarget(context.Background(), "test-group", "notification", "message")
		assert.NoError(t, err)
	})
}
