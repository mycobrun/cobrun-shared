// Package signalr provides Azure SignalR Service integration.
package signalr

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config holds SignalR configuration.
type Config struct {
	ConnectionString string
	HubName          string
}

// Client provides SignalR Service operations.
type Client struct {
	config     Config
	httpClient *http.Client
	endpoint   string
	accessKey  string
}

// NewClient creates a new SignalR client.
func NewClient(config Config) (*Client, error) {
	endpoint, accessKey, err := parseConnectionString(config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   endpoint,
		accessKey:  accessKey,
	}, nil
}

// parseConnectionString parses the SignalR connection string.
func parseConnectionString(connStr string) (endpoint, accessKey string, err error) {
	parts := strings.Split(connStr, ";")
	for _, part := range parts {
		if strings.HasPrefix(part, "Endpoint=") {
			endpoint = strings.TrimPrefix(part, "Endpoint=")
		} else if strings.HasPrefix(part, "AccessKey=") {
			accessKey = strings.TrimPrefix(part, "AccessKey=")
		}
	}

	if endpoint == "" || accessKey == "" {
		return "", "", fmt.Errorf("invalid connection string format")
	}

	return endpoint, accessKey, nil
}

// NegotiateResponse represents the negotiate endpoint response.
type NegotiateResponse struct {
	URL         string `json:"url"`
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"` // Seconds
}

// Negotiate generates a client token for connecting to SignalR.
func (c *Client) Negotiate(ctx context.Context, userID string, groups []string, expiresIn time.Duration) (*NegotiateResponse, error) {
	if expiresIn <= 0 {
		expiresIn = 1 * time.Hour
	}

	// Build the client hub URL
	hubURL := fmt.Sprintf("%s/client/?hub=%s", c.endpoint, c.config.HubName)

	// Generate access token
	accessToken, err := c.generateAccessToken(hubURL, userID, groups, expiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &NegotiateResponse{
		URL:         hubURL,
		AccessToken: accessToken,
		ExpiresIn:   int64(expiresIn.Seconds()),
	}, nil
}

// generateAccessToken generates a JWT-like access token for SignalR.
func (c *Client) generateAccessToken(audience, userID string, groups []string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(expiresIn)

	// Build claims
	claims := map[string]interface{}{
		"aud": audience,
		"iat": now.Unix(),
		"exp": exp.Unix(),
	}

	if userID != "" {
		claims["sub"] = userID
		claims["nameid"] = userID
	}

	if len(groups) > 0 {
		claims["groups"] = groups
	}

	// Build header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	// Encode header
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Encode claims
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signature
	signingInput := headerB64 + "." + claimsB64
	keyBytes, err := base64.StdEncoding.DecodeString(c.accessKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode access key: %w", err)
	}

	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// SendToUser sends a message to a specific user.
func (c *Client) SendToUser(ctx context.Context, userID, target string, arguments []interface{}) error {
	return c.sendMessage(ctx, fmt.Sprintf("/api/v1/hubs/%s/users/%s", c.config.HubName, userID), target, arguments)
}

// SendToGroup sends a message to a group.
func (c *Client) SendToGroup(ctx context.Context, group, target string, arguments []interface{}) error {
	return c.sendMessage(ctx, fmt.Sprintf("/api/v1/hubs/%s/groups/%s", c.config.HubName, group), target, arguments)
}

// Broadcast sends a message to all connections.
func (c *Client) Broadcast(ctx context.Context, target string, arguments []interface{}) error {
	return c.sendMessage(ctx, fmt.Sprintf("/api/v1/hubs/%s", c.config.HubName), target, arguments)
}

// AddToGroup adds a user to a group.
func (c *Client) AddToGroup(ctx context.Context, group, userID string) error {
	path := fmt.Sprintf("/api/v1/hubs/%s/groups/%s/users/%s", c.config.HubName, group, userID)
	return c.makeRequest(ctx, "PUT", path, nil)
}

// RemoveFromGroup removes a user from a group.
func (c *Client) RemoveFromGroup(ctx context.Context, group, userID string) error {
	path := fmt.Sprintf("/api/v1/hubs/%s/groups/%s/users/%s", c.config.HubName, group, userID)
	return c.makeRequest(ctx, "DELETE", path, nil)
}

// sendMessage sends a message to SignalR.
func (c *Client) sendMessage(ctx context.Context, path, target string, arguments []interface{}) error {
	body := map[string]interface{}{
		"target":    target,
		"arguments": arguments,
	}

	return c.makeRequest(ctx, "POST", path, body)
}

// makeRequest makes an authenticated request to SignalR Service.
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}) error {
	// Build URL
	apiURL := c.endpoint + path

	// Marshal body if present
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, apiURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Generate access token for API
	parsedURL, _ := url.Parse(apiURL)
	audience := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
	accessToken, err := c.generateAccessToken(audience, "", nil, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to generate access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("signalr request failed with status %d", resp.StatusCode)
	}

	return nil
}
