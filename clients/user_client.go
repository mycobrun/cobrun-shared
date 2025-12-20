// Package clients provides HTTP clients for service-to-service communication.
package clients

import (
	"context"
	"fmt"
	"time"

	pkghttp "github.com/cobrun/cobrun-platform/pkg/http"
)

// UserClient is an HTTP client for user-related operations.
// This could query a dedicated user service or the auth/identity service.
type UserClient struct {
	client *pkghttp.ResilientClient
}

// UserClientConfig holds configuration for the user client.
type UserClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// DefaultUserClientConfig returns sensible defaults.
func DefaultUserClientConfig(baseURL string) UserClientConfig {
	return UserClientConfig{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
	}
}

// NewUserClient creates a new user service client.
func NewUserClient(config UserClientConfig) *UserClient {
	resilientConfig := pkghttp.DefaultResilientClientConfig("user-service", config.BaseURL)
	resilientConfig.Timeout = config.Timeout

	return &UserClient{
		client: pkghttp.NewResilientClient(resilientConfig),
	}
}

// User represents a user in the system.
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	UserType    string    `json:"user_type"` // rider, driver, admin
	Status      string    `json:"status"`    // active, suspended, deleted
	Rating      float64   `json:"rating,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetUser retrieves a user by ID.
func (c *UserClient) GetUser(ctx context.Context, userID string) (*User, error) {
	var user User
	if err := c.client.GetJSON(ctx, "/api/v1/users/"+userID, &user); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserEmail retrieves a user's email address.
func (c *UserClient) GetUserEmail(ctx context.Context, userID string) (string, error) {
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.Email, nil
}

// GetUserPhone retrieves a user's phone number.
func (c *UserClient) GetUserPhone(ctx context.Context, userID string) (string, error) {
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.Phone, nil
}

// GetUserDisplayName retrieves a user's display name.
func (c *UserClient) GetUserDisplayName(ctx context.Context, userID string) (string, error) {
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return "", err
	}
	if user.DisplayName != "" {
		return user.DisplayName, nil
	}
	return user.FirstName + " " + user.LastName, nil
}

// RecentPlace represents a recently visited place.
type RecentPlace struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	PlaceType string  `json:"place_type"` // home, work, saved, recent
	VisitedAt string  `json:"visited_at"`
}

// GetRecentPlaces retrieves a user's recent places.
func (c *UserClient) GetRecentPlaces(ctx context.Context, userID string, limit int) ([]*RecentPlace, error) {
	var places []*RecentPlace
	path := fmt.Sprintf("/api/v1/users/%s/places/recent?limit=%d", userID, limit)
	if err := c.client.GetJSON(ctx, path, &places); err != nil {
		return nil, fmt.Errorf("failed to get recent places: %w", err)
	}
	return places, nil
}

// SavedPlace represents a user's saved place.
type SavedPlace struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	PlaceType string  `json:"place_type"` // home, work, favorite
	Icon      string  `json:"icon,omitempty"`
}

// GetSavedPlaces retrieves a user's saved places.
func (c *UserClient) GetSavedPlaces(ctx context.Context, userID string) ([]*SavedPlace, error) {
	var places []*SavedPlace
	path := fmt.Sprintf("/api/v1/users/%s/places/saved", userID)
	if err := c.client.GetJSON(ctx, path, &places); err != nil {
		return nil, fmt.Errorf("failed to get saved places: %w", err)
	}
	return places, nil
}

// UpdateUserRating updates a user's rating.
func (c *UserClient) UpdateUserRating(ctx context.Context, userID string, newRating float64) error {
	req := map[string]interface{}{
		"rating": newRating,
	}
	path := fmt.Sprintf("/api/v1/users/%s/rating", userID)
	if err := c.client.PostJSON(ctx, path, req, nil); err != nil {
		return fmt.Errorf("failed to update user rating: %w", err)
	}
	return nil
}

// Health checks if the user service is healthy.
func (c *UserClient) Health(ctx context.Context) error {
	_, err := c.client.Get(ctx, "/health", nil)
	return err
}
