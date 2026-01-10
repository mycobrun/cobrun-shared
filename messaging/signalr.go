// Package messaging provides messaging client utilities.
package messaging

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SignalRConfig holds SignalR Service configuration.
type SignalRConfig struct {
	ConnectionString string
	HubName          string
}

// SignalRClient provides methods to send messages via Azure SignalR Service REST API.
type SignalRClient struct {
	endpoint    string
	accessKey   string
	hubName     string
	httpClient  *http.Client
}

// NewSignalRClient creates a new SignalR client.
func NewSignalRClient(config SignalRConfig) (*SignalRClient, error) {
	endpoint, accessKey, err := parseConnectionString(config.ConnectionString)
	if err != nil {
		return nil, err
	}

	return &SignalRClient{
		endpoint:   endpoint,
		accessKey:  accessKey,
		hubName:    config.HubName,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func parseConnectionString(connStr string) (endpoint, accessKey string, err error) {
	parts := strings.Split(connStr, ";")
	params := make(map[string]string)

	for _, part := range parts {
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}

	endpoint = params["Endpoint"]
	accessKey = params["AccessKey"]

	if endpoint == "" || accessKey == "" {
		return "", "", fmt.Errorf("invalid connection string: missing Endpoint or AccessKey")
	}

	// Ensure endpoint has proper format
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}

	return endpoint, accessKey, nil
}

// generateToken generates an access token for SignalR REST API.
func (c *SignalRClient) generateToken(audience string, expiresAt time.Time) (string, error) {
	// Create claims
	claims := map[string]interface{}{
		"aud": audience,
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
	}

	// Encode header
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	// Encode payload
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)

	// Create signature
	signingInput := header + "." + payloadEncoded
	mac := hmac.New(sha256.New, []byte(c.accessKey))
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// BroadcastMessage broadcasts a message to all clients.
func (c *SignalRClient) BroadcastMessage(ctx context.Context, message *SignalRMessage) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s", c.endpoint, c.hubName)
	return c.sendRequest(ctx, "POST", url, message)
}

// SendToUser sends a message to a specific user.
func (c *SignalRClient) SendToUser(ctx context.Context, userID string, message *SignalRMessage) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/users/%s", c.endpoint, c.hubName, userID)
	return c.sendRequest(ctx, "POST", url, message)
}

// SendToGroup sends a message to a group.
func (c *SignalRClient) SendToGroup(ctx context.Context, groupName string, message *SignalRMessage) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/groups/%s", c.endpoint, c.hubName, groupName)
	return c.sendRequest(ctx, "POST", url, message)
}

// SendToConnection sends a message to a specific connection.
func (c *SignalRClient) SendToConnection(ctx context.Context, connectionID string, message *SignalRMessage) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/connections/%s", c.endpoint, c.hubName, connectionID)
	return c.sendRequest(ctx, "POST", url, message)
}

// AddUserToGroup adds a user to a group.
func (c *SignalRClient) AddUserToGroup(ctx context.Context, userID, groupName string) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/groups/%s/users/%s", c.endpoint, c.hubName, groupName, userID)
	return c.sendRequest(ctx, "PUT", url, nil)
}

// RemoveUserFromGroup removes a user from a group.
func (c *SignalRClient) RemoveUserFromGroup(ctx context.Context, userID, groupName string) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/groups/%s/users/%s", c.endpoint, c.hubName, groupName, userID)
	return c.sendRequest(ctx, "DELETE", url, nil)
}

// AddConnectionToGroup adds a connection to a group.
func (c *SignalRClient) AddConnectionToGroup(ctx context.Context, connectionID, groupName string) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/groups/%s/connections/%s", c.endpoint, c.hubName, groupName, connectionID)
	return c.sendRequest(ctx, "PUT", url, nil)
}

// RemoveConnectionFromGroup removes a connection from a group.
func (c *SignalRClient) RemoveConnectionFromGroup(ctx context.Context, connectionID, groupName string) error {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/groups/%s/connections/%s", c.endpoint, c.hubName, groupName, connectionID)
	return c.sendRequest(ctx, "DELETE", url, nil)
}

// CloseConnection closes a connection.
func (c *SignalRClient) CloseConnection(ctx context.Context, connectionID, reason string) error {
	baseURL := fmt.Sprintf("%s/api/v1/hubs/%s/connections/%s", c.endpoint, c.hubName, connectionID)
	if reason != "" {
		baseURL += "?reason=" + url.QueryEscape(reason)
	}
	return c.sendRequest(ctx, "DELETE", baseURL, nil)
}

// CheckUserExists checks if a user has any connections.
func (c *SignalRClient) CheckUserExists(ctx context.Context, userID string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/users/%s", c.endpoint, c.hubName, userID)

	token, err := c.generateToken(url, time.Now().Add(5*time.Minute))
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// CheckGroupExists checks if a group has any connections.
func (c *SignalRClient) CheckGroupExists(ctx context.Context, groupName string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/hubs/%s/groups/%s", c.endpoint, c.hubName, groupName)

	token, err := c.generateToken(url, time.Now().Add(5*time.Minute))
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

func (c *SignalRClient) sendRequest(ctx context.Context, method, url string, body interface{}) error {
	token, err := c.generateToken(url, time.Now().Add(5*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("signalr request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SignalRMessage represents a message to send via SignalR.
type SignalRMessage struct {
	Target    string        `json:"target"`
	Arguments []interface{} `json:"arguments"`
}

// NewSignalRMessage creates a new SignalR message.
func NewSignalRMessage(target string, args ...interface{}) *SignalRMessage {
	return &SignalRMessage{
		Target:    target,
		Arguments: args,
	}
}

// NegotiateResponse represents the response from negotiate endpoint.
type NegotiateResponse struct {
	URL         string `json:"url"`
	AccessToken string `json:"accessToken"`
}

// GenerateClientToken generates a client access token for direct SignalR connection.
func (c *SignalRClient) GenerateClientToken(userID string, ttl time.Duration) (*NegotiateResponse, error) {
	clientURL := fmt.Sprintf("%s/client/?hub=%s", c.endpoint, c.hubName)

	// Create claims with user ID
	claims := map[string]interface{}{
		"aud": clientURL,
		"exp": time.Now().Add(ttl).Unix(),
		"iat": time.Now().Unix(),
	}

	if userID != "" {
		claims["sub"] = userID
	}

	// Encode header
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	// Encode payload
	payload, err := json.Marshal(claims)
	if err != nil {
		return nil, err
	}
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)

	// Create signature
	signingInput := header + "." + payloadEncoded
	mac := hmac.New(sha256.New, []byte(c.accessKey))
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	token := signingInput + "." + signature

	return &NegotiateResponse{
		URL:         clientURL,
		AccessToken: token,
	}, nil
}

// Common message types for rideshare platform

// LocationUpdateMessage represents a driver location update.
type LocationUpdateMessage struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Heading   float64 `json:"heading"`
	Speed     float64 `json:"speed"`
	Timestamp int64   `json:"timestamp"`
}

// TripStatusMessage represents a trip status update.
type TripStatusMessage struct {
	TripID     string `json:"trip_id"`
	Status     string `json:"status"`
	DriverID   string `json:"driver_id,omitempty"`
	DriverName string `json:"driver_name,omitempty"`
	ETA        int    `json:"eta,omitempty"` // seconds
	Message    string `json:"message,omitempty"`
}

// Helper methods for common rideshare scenarios

// SendLocationUpdate sends a location update to a rider.
func (c *SignalRClient) SendLocationUpdate(ctx context.Context, riderID string, update *LocationUpdateMessage) error {
	msg := NewSignalRMessage("locationUpdate", update)
	return c.SendToUser(ctx, riderID, msg)
}

// SendTripStatusUpdate sends a trip status update to a user.
func (c *SignalRClient) SendTripStatusUpdate(ctx context.Context, userID string, update *TripStatusMessage) error {
	msg := NewSignalRMessage("tripStatus", update)
	return c.SendToUser(ctx, userID, msg)
}

// BroadcastToTrip sends a message to all participants in a trip.
func (c *SignalRClient) BroadcastToTrip(ctx context.Context, tripID string, message *SignalRMessage) error {
	groupName := fmt.Sprintf("trip-%s", tripID)
	return c.SendToGroup(ctx, groupName, message)
}

// AddToTripGroup adds a user to a trip's group.
func (c *SignalRClient) AddToTripGroup(ctx context.Context, tripID, userID string) error {
	groupName := fmt.Sprintf("trip-%s", tripID)
	return c.AddUserToGroup(ctx, userID, groupName)
}

// RemoveFromTripGroup removes a user from a trip's group.
func (c *SignalRClient) RemoveFromTripGroup(ctx context.Context, tripID, userID string) error {
	groupName := fmt.Sprintf("trip-%s", tripID)
	return c.RemoveUserFromGroup(ctx, userID, groupName)
}

// SendToUserWithTarget sends a message to a user with a specific target method.
func (c *SignalRClient) SendToUserWithTarget(ctx context.Context, userID, target string, data interface{}) error {
	msg := NewSignalRMessage(target, data)
	return c.SendToUser(ctx, userID, msg)
}

// SendToGroupWithTarget sends a message to a group with a specific target method.
func (c *SignalRClient) SendToGroupWithTarget(ctx context.Context, groupName, target string, data interface{}) error {
	msg := NewSignalRMessage(target, data)
	return c.SendToGroup(ctx, groupName, msg)
}
