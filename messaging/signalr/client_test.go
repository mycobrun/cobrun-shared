package signalr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.service.signalr.net;AccessKey=testkey",
			HubName:          "test-hub",
		}

		assert.NotEmpty(t, config.ConnectionString)
		assert.Equal(t, "test-hub", config.HubName)
	})

	t.Run("empty hub name", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.service.signalr.net;AccessKey=testkey",
			HubName:          "",
		}

		assert.Empty(t, config.HubName)
	})
}

func TestParseConnectionString(t *testing.T) {
	t.Run("valid connection string", func(t *testing.T) {
		connStr := "Endpoint=https://test.signalr.net;AccessKey=testkey123"
		endpoint, accessKey, err := parseConnectionString(connStr)

		require.NoError(t, err)
		assert.Equal(t, "https://test.signalr.net", endpoint)
		assert.Equal(t, "testkey123", accessKey)
	})

	t.Run("connection string with reversed order", func(t *testing.T) {
		connStr := "AccessKey=testkey456;Endpoint=https://test.signalr.net"
		endpoint, accessKey, err := parseConnectionString(connStr)

		require.NoError(t, err)
		assert.Equal(t, "https://test.signalr.net", endpoint)
		assert.Equal(t, "testkey456", accessKey)
	})

	t.Run("connection string with extra spaces", func(t *testing.T) {
		connStr := "Endpoint=https://test.signalr.net ;AccessKey=testkey "
		endpoint, accessKey, err := parseConnectionString(connStr)

		require.NoError(t, err)
		assert.Equal(t, "https://test.signalr.net ", endpoint)
		assert.Equal(t, "testkey ", accessKey)
	})

	t.Run("missing endpoint", func(t *testing.T) {
		connStr := "AccessKey=testkey"
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid connection string format")
	})

	t.Run("missing access key", func(t *testing.T) {
		connStr := "Endpoint=https://test.signalr.net"
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid connection string format")
	})

	t.Run("empty connection string", func(t *testing.T) {
		connStr := ""
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
	})

	t.Run("malformed connection string", func(t *testing.T) {
		connStr := "InvalidFormat"
		_, _, err := parseConnectionString(connStr)

		assert.Error(t, err)
	})

	t.Run("connection string with additional parameters", func(t *testing.T) {
		connStr := "Endpoint=https://test.signalr.net;AccessKey=testkey;Version=1.0"
		endpoint, accessKey, err := parseConnectionString(connStr)

		require.NoError(t, err)
		assert.Equal(t, "https://test.signalr.net", endpoint)
		assert.Equal(t, "testkey", accessKey)
	})
}

func TestNewClient(t *testing.T) {
	t.Run("create client with valid config", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.signalr.net;AccessKey=dGVzdGtleQ==",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://test.signalr.net", client.endpoint)
		assert.Equal(t, "dGVzdGtleQ==", client.accessKey)
		assert.NotNil(t, client.httpClient)
		assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	})

	t.Run("create client with invalid config", func(t *testing.T) {
		config := Config{
			ConnectionString: "invalid",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to parse connection string")
	})

	t.Run("create client with empty connection string", func(t *testing.T) {
		config := Config{
			ConnectionString: "",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestNegotiateResponse(t *testing.T) {
	t.Run("create negotiate response", func(t *testing.T) {
		resp := &NegotiateResponse{
			URL:         "https://test.signalr.net/client/?hub=test",
			AccessToken: "token123",
			ExpiresIn:   3600,
		}

		assert.NotEmpty(t, resp.URL)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
	})

	t.Run("serialize negotiate response", func(t *testing.T) {
		resp := &NegotiateResponse{
			URL:         "https://test.signalr.net/client/?hub=test",
			AccessToken: "token456",
			ExpiresIn:   7200,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded NegotiateResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.URL, decoded.URL)
		assert.Equal(t, resp.AccessToken, decoded.AccessToken)
		assert.Equal(t, resp.ExpiresIn, decoded.ExpiresIn)
	})
}

func TestNegotiate(t *testing.T) {
	t.Run("negotiate without user ID", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.signalr.net;AccessKey=dGVzdGtleQ==",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		resp, err := client.Negotiate(context.Background(), "", nil, 1*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.URL, "test-hub")
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
	})

	t.Run("negotiate with user ID", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.signalr.net;AccessKey=dGVzdGtleQ==",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		resp, err := client.Negotiate(context.Background(), "user-123", nil, 2*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, int64(7200), resp.ExpiresIn)

		// Verify token structure (should be JWT-like with 3 parts)
		parts := strings.Split(resp.AccessToken, ".")
		assert.Equal(t, 3, len(parts))
	})

	t.Run("negotiate with groups", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.signalr.net;AccessKey=dGVzdGtleQ==",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		groups := []string{"group1", "group2"}
		resp, err := client.Negotiate(context.Background(), "user-456", groups, 1*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
	})

	t.Run("negotiate with zero duration uses default", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.signalr.net;AccessKey=dGVzdGtleQ==",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		resp, err := client.Negotiate(context.Background(), "user-789", nil, 0)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(3600), resp.ExpiresIn) // Default 1 hour
	})

	t.Run("negotiate with negative duration uses default", func(t *testing.T) {
		config := Config{
			ConnectionString: "Endpoint=https://test.signalr.net;AccessKey=dGVzdGtleQ==",
			HubName:          "test-hub",
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		resp, err := client.Negotiate(context.Background(), "user-999", nil, -1*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
	})
}

func TestGenerateAccessToken(t *testing.T) {
	t.Run("generate token without user and groups", func(t *testing.T) {
		client := &Client{
			accessKey: base64.StdEncoding.EncodeToString([]byte("testkey")),
		}

		token, err := client.generateAccessToken("https://test.signalr.net/api", "", nil, 1*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify token structure
		parts := strings.Split(token, ".")
		assert.Equal(t, 3, len(parts))
	})

	t.Run("generate token with user ID", func(t *testing.T) {
		client := &Client{
			accessKey: base64.StdEncoding.EncodeToString([]byte("testkey")),
		}

		token, err := client.generateAccessToken("https://test.signalr.net/api", "user-123", nil, 2*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		parts := strings.Split(token, ".")
		assert.Equal(t, 3, len(parts))

		// Decode and verify claims contain user ID
		claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)

		var claims map[string]interface{}
		err = json.Unmarshal(claimsJSON, &claims)
		require.NoError(t, err)

		assert.Equal(t, "user-123", claims["sub"])
		assert.Equal(t, "user-123", claims["nameid"])
	})

	t.Run("generate token with groups", func(t *testing.T) {
		client := &Client{
			accessKey: base64.StdEncoding.EncodeToString([]byte("testkey")),
		}

		groups := []string{"group1", "group2", "group3"}
		token, err := client.generateAccessToken("https://test.signalr.net/api", "user-456", groups, 1*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Decode and verify claims contain groups
		parts := strings.Split(token, ".")
		claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)

		var claims map[string]interface{}
		err = json.Unmarshal(claimsJSON, &claims)
		require.NoError(t, err)

		groupsArray, ok := claims["groups"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 3, len(groupsArray))
	})

	t.Run("verify token expiration", func(t *testing.T) {
		client := &Client{
			accessKey: base64.StdEncoding.EncodeToString([]byte("testkey")),
		}

		expiresIn := 30 * time.Minute
		token, err := client.generateAccessToken("https://test.signalr.net/api", "", nil, expiresIn)
		require.NoError(t, err)

		parts := strings.Split(token, ".")
		claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)

		var claims map[string]interface{}
		err = json.Unmarshal(claimsJSON, &claims)
		require.NoError(t, err)

		iat, ok := claims["iat"].(float64)
		assert.True(t, ok)

		exp, ok := claims["exp"].(float64)
		assert.True(t, ok)

		// Verify exp is approximately 30 minutes after iat
		diff := int64(exp) - int64(iat)
		assert.InDelta(t, 1800, diff, 5) // 30 minutes = 1800 seconds, with 5 second tolerance
	})
}

func TestSendToUser(t *testing.T) {
	t.Run("send message to user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/api/v1/hubs/test-hub/users/user-123")
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Read and verify body
			var body map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, "testMethod", body["target"])

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			config:     Config{HubName: "test-hub"},
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.SendToUser(context.Background(), "user-123", "testMethod", []interface{}{"arg1", 42})
		assert.NoError(t, err)
	})

	t.Run("send message with error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client := &Client{
			config:     Config{HubName: "test-hub"},
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.SendToUser(context.Background(), "user-123", "testMethod", []interface{}{})
		assert.Error(t, err)
	})
}

func TestSendToGroup(t *testing.T) {
	t.Run("send message to group", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/api/v1/hubs/test-hub/groups/test-group")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			config:     Config{HubName: "test-hub"},
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.SendToGroup(context.Background(), "test-group", "notification", []interface{}{"message"})
		assert.NoError(t, err)
	})
}

func TestBroadcast(t *testing.T) {
	t.Run("broadcast message to all", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/api/v1/hubs/test-hub")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			config:     Config{HubName: "test-hub"},
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.Broadcast(context.Background(), "globalUpdate", []interface{}{"data"})
		assert.NoError(t, err)
	})
}

func TestAddToGroup(t *testing.T) {
	t.Run("add user to group", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Contains(t, r.URL.Path, "/api/v1/hubs/test-hub/groups/group-1/users/user-123")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			config:     Config{HubName: "test-hub"},
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.AddToGroup(context.Background(), "group-1", "user-123")
		assert.NoError(t, err)
	})
}

func TestRemoveFromGroup(t *testing.T) {
	t.Run("remove user from group", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)
			assert.Contains(t, r.URL.Path, "/api/v1/hubs/test-hub/groups/group-2/users/user-456")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			config:     Config{HubName: "test-hub"},
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.RemoveFromGroup(context.Background(), "group-2", "user-456")
		assert.NoError(t, err)
	})
}

func TestMakeRequest(t *testing.T) {
	t.Run("make request with body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

			var body map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, "value", body["key"])

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		body := map[string]interface{}{
			"key": "value",
		}

		err := client.makeRequest(context.Background(), "POST", "/test", body)
		assert.NoError(t, err)
	})

	t.Run("make request without body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)

			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(r.Body)
			assert.Empty(t, buf.String())

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.makeRequest(context.Background(), "DELETE", "/test", nil)
		assert.NoError(t, err)
	})

	t.Run("make request with server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := &Client{
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		err := client.makeRequest(context.Background(), "POST", "/test", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("make request with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			endpoint:   server.URL,
			accessKey:  base64.StdEncoding.EncodeToString([]byte("testkey")),
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := client.makeRequest(ctx, "POST", "/test", nil)
		assert.Error(t, err)
	})
}
