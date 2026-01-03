// Package clients provides HTTP clients for service-to-service communication.
package clients

import (
	"context"
	"fmt"
	"time"

	pkghttp "github.com/mycobrun/cobrun-shared/http"
)

// NotificationClient is an HTTP client for the notifications service.
type NotificationClient struct {
	client *pkghttp.ResilientClient
}

// NotificationClientConfig holds configuration for the notification client.
type NotificationClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// DefaultNotificationClientConfig returns sensible defaults.
func DefaultNotificationClientConfig(baseURL string) NotificationClientConfig {
	return NotificationClientConfig{
		BaseURL: baseURL,
		Timeout: 10 * time.Second,
	}
}

// NewNotificationClient creates a new notification service client.
func NewNotificationClient(config NotificationClientConfig) *NotificationClient {
	resilientConfig := pkghttp.DefaultResilientClientConfig("notification-service", config.BaseURL)
	resilientConfig.Timeout = config.Timeout

	return &NotificationClient{
		client: pkghttp.NewResilientClient(resilientConfig),
	}
}

// SendNotificationRequest represents a request to send a notification.
type SendNotificationRequest struct {
	UserID      string            `json:"user_id"`
	Type        string            `json:"type"`
	Channels    []string          `json:"channels,omitempty"`
	Title       string            `json:"title,omitempty"`
	Body        string            `json:"body,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	TemplateID  string            `json:"template_id,omitempty"`
	ReferenceID string            `json:"reference_id,omitempty"`
	Priority    string            `json:"priority,omitempty"`
}

// SendNotificationResponse represents the response from sending a notification.
type SendNotificationResponse struct {
	Notifications []struct {
		ID      string `json:"id"`
		Channel string `json:"channel"`
		Status  string `json:"status"`
	} `json:"notifications"`
}

// Send sends a notification to a user.
func (c *NotificationClient) Send(ctx context.Context, req *SendNotificationRequest) (*SendNotificationResponse, error) {
	var resp SendNotificationResponse
	if err := c.client.PostJSON(ctx, "/api/v1/notifications", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}
	return &resp, nil
}

// SendBulkNotificationRequest represents a bulk notification request.
type SendBulkNotificationRequest struct {
	UserIDs    []string          `json:"user_ids"`
	Type       string            `json:"type"`
	Channels   []string          `json:"channels,omitempty"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Data       map[string]string `json:"data,omitempty"`
	TemplateID string            `json:"template_id,omitempty"`
}

// SendBulk sends notifications to multiple users.
func (c *NotificationClient) SendBulk(ctx context.Context, req *SendBulkNotificationRequest) error {
	if err := c.client.PostJSON(ctx, "/api/v1/notifications/bulk", req, nil); err != nil {
		return fmt.Errorf("failed to send bulk notification: %w", err)
	}
	return nil
}

// NotifyTripDriverAssigned sends a driver assigned notification.
func (c *NotificationClient) NotifyTripDriverAssigned(ctx context.Context, userID, tripID, driverName string, eta int) error {
	_, err := c.Send(ctx, &SendNotificationRequest{
		UserID: userID,
		Type:   "trip.driver_assigned",
		Title:  "Driver Assigned",
		Body:   fmt.Sprintf("%s is on the way. ETA: %d minutes", driverName, eta),
		Data: map[string]string{
			"trip_id":     tripID,
			"driver_name": driverName,
			"eta_minutes": fmt.Sprintf("%d", eta),
		},
		Priority:    "high",
		ReferenceID: tripID,
	})
	return err
}

// NotifyTripStarted sends a trip started notification.
func (c *NotificationClient) NotifyTripStarted(ctx context.Context, userID, tripID string) error {
	_, err := c.Send(ctx, &SendNotificationRequest{
		UserID:      userID,
		Type:        "trip.started",
		Title:       "Trip Started",
		Body:        "Your trip has started. Enjoy your ride!",
		Data:        map[string]string{"trip_id": tripID},
		Priority:    "normal",
		ReferenceID: tripID,
	})
	return err
}

// NotifyTripCompleted sends a trip completed notification.
func (c *NotificationClient) NotifyTripCompleted(ctx context.Context, userID, tripID string, fare float64) error {
	_, err := c.Send(ctx, &SendNotificationRequest{
		UserID:   userID,
		Type:     "trip.completed",
		Title:    "Trip Completed",
		Body:     fmt.Sprintf("Thanks for riding! Your fare: $%.2f", fare),
		Data:     map[string]string{"trip_id": tripID, "fare": fmt.Sprintf("%.2f", fare)},
		Priority: "normal",
	})
	return err
}

// NotifyNewRideOffer sends a new ride offer notification to a driver.
func (c *NotificationClient) NotifyNewRideOffer(ctx context.Context, driverID, tripID string, pickupAddress string, fare float64) error {
	_, err := c.Send(ctx, &SendNotificationRequest{
		UserID:   driverID,
		Type:     "driver.new_ride_offer",
		Title:    "New Ride Request",
		Body:     fmt.Sprintf("Pickup: %s - Estimated fare: $%.2f", pickupAddress, fare),
		Data:     map[string]string{"trip_id": tripID, "pickup": pickupAddress},
		Priority: "high",
	})
	return err
}

// NotifyPaymentSuccess sends a payment success notification.
func (c *NotificationClient) NotifyPaymentSuccess(ctx context.Context, userID, paymentID string, amount float64) error {
	_, err := c.Send(ctx, &SendNotificationRequest{
		UserID:   userID,
		Type:     "payment.success",
		Title:    "Payment Successful",
		Body:     fmt.Sprintf("Your payment of $%.2f was successful", amount),
		Data:     map[string]string{"payment_id": paymentID, "amount": fmt.Sprintf("%.2f", amount)},
		Priority: "normal",
	})
	return err
}

// NotifySupportTicketUpdate sends a support ticket update notification.
func (c *NotificationClient) NotifySupportTicketUpdate(ctx context.Context, userID, ticketID, message string) error {
	_, err := c.Send(ctx, &SendNotificationRequest{
		UserID:      userID,
		Type:        "support.ticket_update",
		Title:       "Support Update",
		Body:        message,
		Data:        map[string]string{"ticket_id": ticketID},
		Priority:    "normal",
		ReferenceID: ticketID,
	})
	return err
}

// Health checks if the notification service is healthy.
func (c *NotificationClient) Health(ctx context.Context) error {
	_, err := c.client.Get(ctx, "/health", nil)
	return err
}
