// Package clients provides HTTP clients for service-to-service communication.
package clients

import (
	"context"
	"fmt"
	"time"

	pkghttp "github.com/cobrun/cobrun-platform/pkg/http"
)

// PricingClient is an HTTP client for the pricing service.
type PricingClient struct {
	client *pkghttp.ResilientClient
}

// PricingClientConfig holds configuration for the pricing client.
type PricingClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// DefaultPricingClientConfig returns sensible defaults.
func DefaultPricingClientConfig(baseURL string) PricingClientConfig {
	return PricingClientConfig{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
	}
}

// NewPricingClient creates a new pricing service client.
func NewPricingClient(config PricingClientConfig) *PricingClient {
	resilientConfig := pkghttp.DefaultResilientClientConfig("pricing-service", config.BaseURL)
	resilientConfig.Timeout = config.Timeout

	return &PricingClient{
		client: pkghttp.NewResilientClient(resilientConfig),
	}
}

// Point represents a geographic point.
type Point struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// PriceEstimateRequest represents a request for a price estimate.
type PriceEstimateRequest struct {
	PickupLocation  Point   `json:"pickup_location"`
	DropoffLocation Point   `json:"dropoff_location"`
	RideType        string  `json:"ride_type"`
	ScheduledAt     *string `json:"scheduled_at,omitempty"`
	PromoCode       string  `json:"promo_code,omitempty"`
	UserID          string  `json:"user_id,omitempty"`
}

// PriceEstimate represents a price estimate response.
type PriceEstimate struct {
	RideType                string         `json:"ride_type"`
	DistanceMeters          float64        `json:"distance_meters"`
	DistanceKm              float64        `json:"distance_km"`
	DurationSeconds         int            `json:"duration_seconds"`
	DurationMinutes         float64        `json:"duration_minutes"`
	SurgeMultiplier         float64        `json:"surge_multiplier"`
	SurgeZone               string         `json:"surge_zone,omitempty"`
	EstimatedFare           float64        `json:"estimated_fare"`
	EstimatedFareAfterPromo float64        `json:"estimated_fare_after_promo,omitempty"`
	MinFare                 float64        `json:"min_fare"`
	MaxFare                 float64        `json:"max_fare"`
	Currency                string         `json:"currency"`
	FareBreakdown           *FareBreakdown `json:"fare_breakdown"`
	PromoApplied            bool           `json:"promo_applied"`
	PromoCode               string         `json:"promo_code,omitempty"`
	PromoDiscount           float64        `json:"promo_discount,omitempty"`
	PromoMessage            string         `json:"promo_message,omitempty"`
	ValidUntil              time.Time      `json:"valid_until"`
}

// FareBreakdown represents the breakdown of a fare.
type FareBreakdown struct {
	BaseFare      float64 `json:"base_fare"`
	DistanceFare  float64 `json:"distance_fare"`
	TimeFare      float64 `json:"time_fare"`
	SurgeFare     float64 `json:"surge_fare,omitempty"`
	TollFees      float64 `json:"toll_fees,omitempty"`
	AirportFee    float64 `json:"airport_fee,omitempty"`
	ServiceFee    float64 `json:"service_fee"`
	PromoDiscount float64 `json:"promo_discount,omitempty"`
	Subtotal      float64 `json:"subtotal"`
	Total         float64 `json:"total"`
}

// FareCalculationRequest represents a request to calculate final fare.
type FareCalculationRequest struct {
	TripID            string  `json:"trip_id,omitempty"`
	RideType          string  `json:"ride_type"`
	DistanceMeters    float64 `json:"distance_meters,omitempty"`
	ActualDistanceKm  float64 `json:"actual_distance_km,omitempty"`
	DurationSeconds   int     `json:"duration_seconds,omitempty"`
	ActualDurationMin int     `json:"actual_duration_min,omitempty"`
	SurgeMultiplier   float64 `json:"surge_multiplier"`
	PromoCode         string  `json:"promo_code,omitempty"`
	UserID            string  `json:"user_id,omitempty"`
}

// FareCalculation represents the calculated fare.
type FareCalculation struct {
	RideType        string         `json:"ride_type"`
	DistanceMeters  float64        `json:"distance_meters"`
	DurationSeconds int            `json:"duration_seconds"`
	SurgeMultiplier float64        `json:"surge_multiplier"`
	TotalFare       float64        `json:"total_fare"`
	Currency        string         `json:"currency"`
	FareBreakdown   *FareBreakdown `json:"fare_breakdown"`
}

// GetEstimate gets a price estimate for a single ride type.
func (c *PricingClient) GetEstimate(ctx context.Context, req *PriceEstimateRequest) (*PriceEstimate, error) {
	var resp PriceEstimate
	if err := c.client.PostJSON(ctx, "/api/v1/pricing/estimate", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get price estimate: %w", err)
	}
	return &resp, nil
}

// MultiEstimateResponse represents estimates for all ride types.
type MultiEstimateResponse struct {
	Estimates    []*PriceEstimate `json:"estimates"`
	PickupETA    int              `json:"pickup_eta_seconds,omitempty"`
	CalculatedAt time.Time        `json:"calculated_at"`
}

// GetMultiEstimate gets price estimates for all ride types.
func (c *PricingClient) GetMultiEstimate(ctx context.Context, pickup, dropoff Point) (*MultiEstimateResponse, error) {
	req := map[string]interface{}{
		"pickup_location":  pickup,
		"dropoff_location": dropoff,
	}

	var resp MultiEstimateResponse
	if err := c.client.PostJSON(ctx, "/api/v1/pricing/estimate/all", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get multi estimate: %w", err)
	}
	return &resp, nil
}

// CalculateFinalFare calculates the final fare for a completed trip.
func (c *PricingClient) CalculateFinalFare(ctx context.Context, req *FareCalculationRequest) (*FareCalculation, error) {
	var resp FareCalculation
	if err := c.client.PostJSON(ctx, "/api/v1/pricing/calculate", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to calculate final fare: %w", err)
	}
	return &resp, nil
}

// GetSurgeMultiplier gets the current surge multiplier for a location.
func (c *PricingClient) GetSurgeMultiplier(ctx context.Context, location Point) (float64, string, error) {
	req := map[string]interface{}{
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
	}

	var resp struct {
		Multiplier float64 `json:"multiplier"`
		ZoneName   string  `json:"zone_name"`
	}
	if err := c.client.PostJSON(ctx, "/api/v1/pricing/surge", req, &resp); err != nil {
		return 1.0, "", fmt.Errorf("failed to get surge multiplier: %w", err)
	}
	return resp.Multiplier, resp.ZoneName, nil
}

// Health checks if the pricing service is healthy.
func (c *PricingClient) Health(ctx context.Context) error {
	_, err := c.client.Get(ctx, "/health", nil)
	return err
}
