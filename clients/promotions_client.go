// Package clients provides HTTP clients for service-to-service communication.
package clients

import (
	"context"
	"fmt"
	"time"

	pkghttp "github.com/mycobrun/cobrun-shared/http"
)

// PromotionsClient is an HTTP client for the promotions service.
type PromotionsClient struct {
	client *pkghttp.ResilientClient
}

// PromotionsClientConfig holds configuration for the promotions client.
type PromotionsClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// DefaultPromotionsClientConfig returns sensible defaults.
func DefaultPromotionsClientConfig(baseURL string) PromotionsClientConfig {
	return PromotionsClientConfig{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
	}
}

// NewPromotionsClient creates a new promotions service client.
func NewPromotionsClient(config PromotionsClientConfig) *PromotionsClient {
	resilientConfig := pkghttp.DefaultResilientClientConfig("promotions-service", config.BaseURL)
	resilientConfig.Timeout = config.Timeout

	return &PromotionsClient{
		client: pkghttp.NewResilientClient(resilientConfig),
	}
}

// PromoValidationRequest represents a request to validate a promo code.
type PromoValidationRequest struct {
	Code       string  `json:"code"`
	UserID     string  `json:"user_id"`
	FareAmount float64 `json:"fare_amount"`
}

// PromoValidation represents the result of promo code validation.
type PromoValidation struct {
	Valid          bool      `json:"valid"`
	PromoID        string    `json:"promo_id,omitempty"`
	Code           string    `json:"code"`
	DiscountType   string    `json:"discount_type,omitempty"`   // percentage, fixed
	DiscountValue  float64   `json:"discount_value,omitempty"`  // Percentage (0-100) or fixed amount
	MaxDiscount    float64   `json:"max_discount,omitempty"`    // Maximum discount for percentage
	DiscountAmount float64   `json:"discount_amount,omitempty"` // Calculated discount for this fare
	Message        string    `json:"message"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`
	RemainingUses  int       `json:"remaining_uses,omitempty"`
}

// ValidatePromo validates a promo code for a user and fare amount.
func (c *PromotionsClient) ValidatePromo(ctx context.Context, code, userID string, fareAmount float64) (*PromoValidation, error) {
	req := &PromoValidationRequest{
		Code:       code,
		UserID:     userID,
		FareAmount: fareAmount,
	}

	var resp PromoValidation
	if err := c.client.PostJSON(ctx, "/api/v1/promotions/validate", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to validate promo: %w", err)
	}
	return &resp, nil
}

// ApplyPromoRequest represents a request to apply a promo code.
type ApplyPromoRequest struct {
	Code     string  `json:"code"`
	UserID   string  `json:"user_id"`
	TripID   string  `json:"trip_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// ApplyPromo applies a promo code to a trip.
func (c *PromotionsClient) ApplyPromo(ctx context.Context, req *ApplyPromoRequest) (*PromoValidation, error) {
	var resp PromoValidation
	if err := c.client.PostJSON(ctx, "/api/v1/promotions/apply", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to apply promo: %w", err)
	}
	return &resp, nil
}

// Promotion represents a promotion.
type Promotion struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	Type           string    `json:"type"` // percentage, fixed, free_ride
	Value          float64   `json:"value"`
	MaxDiscount    float64   `json:"max_discount,omitempty"`
	MinOrderAmount float64   `json:"min_order_amount,omitempty"`
	MaxUses        int       `json:"max_uses,omitempty"`
	UsedCount      int       `json:"used_count"`
	StartsAt       time.Time `json:"starts_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	IsActive       bool      `json:"is_active"`
	Description    string    `json:"description"`
}

// GetPromotion retrieves a promotion by code.
func (c *PromotionsClient) GetPromotion(ctx context.Context, code string) (*Promotion, error) {
	var promo Promotion
	if err := c.client.GetJSON(ctx, "/api/v1/promotions/code/"+code, &promo); err != nil {
		return nil, fmt.Errorf("failed to get promotion: %w", err)
	}
	return &promo, nil
}

// GetUserPromotions retrieves available promotions for a user.
func (c *PromotionsClient) GetUserPromotions(ctx context.Context, userID string) ([]*Promotion, error) {
	var promos []*Promotion
	path := fmt.Sprintf("/api/v1/promotions/user/%s", userID)
	if err := c.client.GetJSON(ctx, path, &promos); err != nil {
		return nil, fmt.Errorf("failed to get user promotions: %w", err)
	}
	return promos, nil
}

// ReferralInfo represents referral program information.
type ReferralInfo struct {
	ReferralCode    string  `json:"referral_code"`
	ReferrerBonus   float64 `json:"referrer_bonus"`
	RefereeDiscount float64 `json:"referee_discount"`
	TotalReferrals  int     `json:"total_referrals"`
	TotalEarnings   float64 `json:"total_earnings"`
}

// GetReferralInfo gets a user's referral program info.
func (c *PromotionsClient) GetReferralInfo(ctx context.Context, userID string) (*ReferralInfo, error) {
	var info ReferralInfo
	path := fmt.Sprintf("/api/v1/referrals/user/%s", userID)
	if err := c.client.GetJSON(ctx, path, &info); err != nil {
		return nil, fmt.Errorf("failed to get referral info: %w", err)
	}
	return &info, nil
}

// Health checks if the promotions service is healthy.
func (c *PromotionsClient) Health(ctx context.Context) error {
	_, err := c.client.Get(ctx, "/health", nil)
	return err
}
