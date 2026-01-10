// Package maps provides a Google Maps Platform adapter for geocoding, routing, and places.
// This adapter is designed to be used server-side only via BFF endpoints.
// Client keys should never be exposed in server code - use the web/mobile SDKs directly.
package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mycobrun/cobrun-shared/geo"
	"github.com/mycobrun/cobrun-shared/logging"
)

const (
	// Google Maps Platform API endpoints
	placesAutocompleteURL = "https://places.googleapis.com/v1/places:autocomplete"
	placeDetailsURL       = "https://places.googleapis.com/v1/places/%s"
	computeRoutesURL      = "https://routes.googleapis.com/directions/v2:computeRoutes"
	computeMatrixURL      = "https://routes.googleapis.com/distanceMatrix/v2:computeRouteMatrix"
	geocodeURL            = "https://maps.googleapis.com/maps/api/geocode/json"

	// Default configuration
	defaultTimeout         = 10 * time.Second
	defaultMaxRetries      = 3
	defaultRetryDelay      = 100 * time.Millisecond
	defaultCacheTTL        = 30 * 24 * time.Hour // 30 days (Google ToS max)
	defaultRateLimitPerSec = 50                  // Conservative default
)

// TravelMode specifies the travel mode for routing.
type TravelMode string

const (
	TravelModeDrive         TravelMode = "DRIVE"
	TravelModeWalk          TravelMode = "WALK"
	TravelModeBicycle       TravelMode = "BICYCLE"
	TravelModeTwoWheeler    TravelMode = "TWO_WHEELER"
	TravelModeTransit       TravelMode = "TRANSIT"
)

// RoutingPreference specifies the routing preference.
type RoutingPreference string

const (
	RoutingPreferenceTrafficUnaware       RoutingPreference = "TRAFFIC_UNAWARE"
	RoutingPreferenceTrafficAware         RoutingPreference = "TRAFFIC_AWARE"
	RoutingPreferenceTrafficAwareOptimal  RoutingPreference = "TRAFFIC_AWARE_OPTIMAL"
)

// Config holds Google Maps adapter configuration.
type Config struct {
	// APIKey is the server-side API key (IP-restricted to NAT Gateway IPs)
	APIKey string

	// Timeout for HTTP requests
	Timeout time.Duration

	// MaxRetries for failed requests
	MaxRetries int

	// RetryDelay between retries
	RetryDelay time.Duration

	// CacheTTL for caching results (max 30 days per Google ToS)
	CacheTTL time.Duration

	// RateLimitPerSecond for throttling
	RateLimitPerSecond int

	// DefaultCountry for autocomplete bias (ISO 3166-1 alpha-2)
	DefaultCountry string

	// DefaultRegion for autocomplete bias
	DefaultRegion string

	// EnableTrafficRouting enables traffic-aware routing (costs more)
	EnableTrafficRouting bool
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig(apiKey string) *Config {
	return &Config{
		APIKey:               apiKey,
		Timeout:              defaultTimeout,
		MaxRetries:           defaultMaxRetries,
		RetryDelay:           defaultRetryDelay,
		CacheTTL:             defaultCacheTTL,
		RateLimitPerSecond:   defaultRateLimitPerSec,
		DefaultCountry:       "US",
		DefaultRegion:        "WA",
		EnableTrafficRouting: true,
	}
}

// Client is the Google Maps Platform client.
type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *logging.Logger
	tracer     *Tracer
	cache      Cache
	limiter    RateLimiter
}

// Cache interface for caching maps responses.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// RateLimiter interface for rate limiting.
type RateLimiter interface {
	Allow(ctx context.Context, key string) bool
	Wait(ctx context.Context, key string) error
}

// NewClient creates a new Google Maps client.
func NewClient(config *Config, logger *logging.Logger, tracer *Tracer, cache Cache, limiter RateLimiter) *Client {
	if config == nil {
		config = DefaultConfig("")
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger:  logger,
		tracer:  tracer,
		cache:   cache,
		limiter: limiter,
	}
}

// === Places Autocomplete ===

// AutocompleteRequest represents a places autocomplete request.
type AutocompleteRequest struct {
	Input        string    // The search text
	SessionToken string    // Session token for billing (group autocomplete + place details)
	Country      string    // ISO 3166-1 alpha-2 country code
	Region       string    // Region bias
	Location     geo.Point // Location bias
	RadiusMeters float64   // Bias radius in meters
	Types        []string  // Place type filter (e.g., "address", "establishment")
}

// AutocompleteResult represents an autocomplete prediction.
type AutocompleteResult struct {
	PlaceID          string `json:"place_id"`
	Description      string `json:"description"`
	MainText         string `json:"main_text"`
	SecondaryText    string `json:"secondary_text"`
	StructuredFormat struct {
		MainText      string `json:"main_text"`
		SecondaryText string `json:"secondary_text"`
	} `json:"structured_format"`
	Types []string `json:"types"`
}

// Autocomplete performs a places autocomplete search.
func (c *Client) Autocomplete(ctx context.Context, req *AutocompleteRequest) ([]AutocompleteResult, error) {
	ctx, span := c.startSpan(ctx, "maps.Autocomplete")
	defer span.End()

	// Rate limiting
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx, "maps:autocomplete"); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Build request body for new Places API
	body := map[string]interface{}{
		"input": req.Input,
	}

	// Add session token if provided
	if req.SessionToken != "" {
		body["sessionToken"] = req.SessionToken
	}

	// Location bias
	if req.Location.Lat != 0 || req.Location.Lng != 0 {
		body["locationBias"] = map[string]interface{}{
			"circle": map[string]interface{}{
				"center": map[string]interface{}{
					"latitude":  req.Location.Lat,
					"longitude": req.Location.Lng,
				},
				"radius": req.RadiusMeters,
			},
		}
	}

	// Country restriction
	country := req.Country
	if country == "" {
		country = c.config.DefaultCountry
	}
	if country != "" {
		body["includedRegionCodes"] = []string{country}
	}

	// Types filter
	if len(req.Types) > 0 {
		body["includedPrimaryTypes"] = req.Types
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", placesAutocompleteURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Goog-Api-Key", c.config.APIKey)
	httpReq.Header.Set("X-Goog-FieldMask", "suggestions.placePrediction.placeId,suggestions.placePrediction.text,suggestions.placePrediction.structuredFormat,suggestions.placePrediction.types")

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Suggestions []struct {
			PlacePrediction struct {
				PlaceID          string `json:"placeId"`
				Text             struct {
					Text string `json:"text"`
				} `json:"text"`
				StructuredFormat struct {
					MainText      struct{ Text string } `json:"mainText"`
					SecondaryText struct{ Text string } `json:"secondaryText"`
				} `json:"structuredFormat"`
				Types []string `json:"types"`
			} `json:"placePrediction"`
		} `json:"suggestions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]AutocompleteResult, 0, len(apiResp.Suggestions))
	for _, s := range apiResp.Suggestions {
		p := s.PlacePrediction
		results = append(results, AutocompleteResult{
			PlaceID:       p.PlaceID,
			Description:   p.Text.Text,
			MainText:      p.StructuredFormat.MainText.Text,
			SecondaryText: p.StructuredFormat.SecondaryText.Text,
			Types:         p.Types,
		})
	}

	c.logger.Debug("autocomplete completed",
		"input", req.Input,
		"results", len(results))

	return results, nil
}

// === Place Details ===

// PlaceDetailsRequest represents a place details request.
type PlaceDetailsRequest struct {
	PlaceID      string
	SessionToken string // Same session token used in autocomplete
	Fields       []string
}

// PlaceDetails represents place details.
type PlaceDetails struct {
	PlaceID          string    `json:"place_id"`
	Name             string    `json:"name"`
	FormattedAddress string    `json:"formatted_address"`
	Location         geo.Point `json:"location"`
	Types            []string  `json:"types"`
	AddressComponents []AddressComponent `json:"address_components"`
}

// AddressComponent represents an address component.
type AddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

// GetPlaceDetails retrieves details for a place.
func (c *Client) GetPlaceDetails(ctx context.Context, req *PlaceDetailsRequest) (*PlaceDetails, error) {
	ctx, span := c.startSpan(ctx, "maps.GetPlaceDetails")
	defer span.End()

	// Check cache first
	cacheKey := fmt.Sprintf("place:%s", req.PlaceID)
	if c.cache != nil {
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			var details PlaceDetails
			if err := json.Unmarshal(cached, &details); err == nil {
				c.logger.Debug("place details cache hit", "placeID", req.PlaceID)
				return &details, nil
			}
		}
	}

	// Rate limiting
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx, "maps:place_details"); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Build URL
	reqURL := fmt.Sprintf(placeDetailsURL, req.PlaceID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Field mask for what data to return
	fields := req.Fields
	if len(fields) == 0 {
		fields = []string{"id", "displayName", "formattedAddress", "location", "types", "addressComponents"}
	}
	httpReq.Header.Set("X-Goog-Api-Key", c.config.APIKey)
	httpReq.Header.Set("X-Goog-FieldMask", strings.Join(fields, ","))

	// Add session token if provided
	if req.SessionToken != "" {
		q := httpReq.URL.Query()
		q.Set("sessionToken", req.SessionToken)
		httpReq.URL.RawQuery = q.Encode()
	}

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		ID               string `json:"id"`
		DisplayName      struct{ Text string } `json:"displayName"`
		FormattedAddress string `json:"formattedAddress"`
		Location         struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
		Types             []string `json:"types"`
		AddressComponents []struct {
			LongText  string   `json:"longText"`
			ShortText string   `json:"shortText"`
			Types     []string `json:"types"`
		} `json:"addressComponents"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	details := &PlaceDetails{
		PlaceID:          apiResp.ID,
		Name:             apiResp.DisplayName.Text,
		FormattedAddress: apiResp.FormattedAddress,
		Location: geo.Point{
			Lat: apiResp.Location.Latitude,
			Lng: apiResp.Location.Longitude,
		},
		Types: apiResp.Types,
	}

	for _, comp := range apiResp.AddressComponents {
		details.AddressComponents = append(details.AddressComponents, AddressComponent{
			LongName:  comp.LongText,
			ShortName: comp.ShortText,
			Types:     comp.Types,
		})
	}

	// Cache the result
	if c.cache != nil {
		if cached, err := json.Marshal(details); err == nil {
			_ = c.cache.Set(ctx, cacheKey, cached, c.config.CacheTTL)
		}
	}

	c.logger.Debug("place details retrieved", "placeID", req.PlaceID)

	return details, nil
}

// === Routes ===

// ComputeRoutesRequest represents a route computation request.
type ComputeRoutesRequest struct {
	Origin             geo.Point
	Destination        geo.Point
	TravelMode         TravelMode
	RoutingPreference  RoutingPreference
	DepartureTime      *time.Time
	AvoidTolls         bool
	AvoidHighways      bool
	AvoidFerries       bool
	ComputeAlternatives bool
}

// RouteResult represents a computed route.
type RouteResult struct {
	DistanceMeters         int           `json:"distance_meters"`
	DurationSeconds        int           `json:"duration_seconds"`
	DurationInTraffic      int           `json:"duration_in_traffic_seconds,omitempty"`
	EncodedPolyline        string        `json:"encoded_polyline"`
	DepartureTime          time.Time     `json:"departure_time"`
	ArrivalTime            time.Time     `json:"arrival_time"`
	StaticDuration         int           `json:"static_duration_seconds"`
	TrafficDelay           int           `json:"traffic_delay_seconds,omitempty"`
	Legs                   []RouteLeg    `json:"legs,omitempty"`
}

// RouteLeg represents a leg of a route.
type RouteLeg struct {
	DistanceMeters    int       `json:"distance_meters"`
	DurationSeconds   int       `json:"duration_seconds"`
	StartLocation     geo.Point `json:"start_location"`
	EndLocation       geo.Point `json:"end_location"`
}

// ComputeRoutes computes a route between origin and destination.
func (c *Client) ComputeRoutes(ctx context.Context, req *ComputeRoutesRequest) (*RouteResult, error) {
	ctx, span := c.startSpan(ctx, "maps.ComputeRoutes")
	defer span.End()

	// Rate limiting
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx, "maps:compute_routes"); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Build request body
	body := map[string]interface{}{
		"origin": map[string]interface{}{
			"location": map[string]interface{}{
				"latLng": map[string]interface{}{
					"latitude":  req.Origin.Lat,
					"longitude": req.Origin.Lng,
				},
			},
		},
		"destination": map[string]interface{}{
			"location": map[string]interface{}{
				"latLng": map[string]interface{}{
					"latitude":  req.Destination.Lat,
					"longitude": req.Destination.Lng,
				},
			},
		},
		"travelMode": req.TravelMode,
	}

	// Routing preference
	preference := req.RoutingPreference
	if preference == "" {
		if c.config.EnableTrafficRouting {
			preference = RoutingPreferenceTrafficAware
		} else {
			preference = RoutingPreferenceTrafficUnaware
		}
	}
	body["routingPreference"] = preference

	// Departure time
	if req.DepartureTime != nil {
		body["departureTime"] = req.DepartureTime.Format(time.RFC3339)
	}

	// Route modifiers
	modifiers := make(map[string]interface{})
	if req.AvoidTolls {
		modifiers["avoidTolls"] = true
	}
	if req.AvoidHighways {
		modifiers["avoidHighways"] = true
	}
	if req.AvoidFerries {
		modifiers["avoidFerries"] = true
	}
	if len(modifiers) > 0 {
		body["routeModifiers"] = modifiers
	}

	body["computeAlternativeRoutes"] = req.ComputeAlternatives

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", computeRoutesURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Goog-Api-Key", c.config.APIKey)
	httpReq.Header.Set("X-Goog-FieldMask", "routes.duration,routes.distanceMeters,routes.polyline.encodedPolyline,routes.legs,routes.staticDuration,routes.travelAdvisory.speedReadingIntervals")

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Routes []struct {
			DistanceMeters int    `json:"distanceMeters"`
			Duration       string `json:"duration"`
			StaticDuration string `json:"staticDuration"`
			Polyline       struct {
				EncodedPolyline string `json:"encodedPolyline"`
			} `json:"polyline"`
			Legs []struct {
				DistanceMeters int    `json:"distanceMeters"`
				Duration       string `json:"duration"`
				StartLocation  struct {
					LatLng struct {
						Latitude  float64 `json:"latitude"`
						Longitude float64 `json:"longitude"`
					} `json:"latLng"`
				} `json:"startLocation"`
				EndLocation struct {
					LatLng struct {
						Latitude  float64 `json:"latitude"`
						Longitude float64 `json:"longitude"`
					} `json:"latLng"`
				} `json:"endLocation"`
			} `json:"legs"`
		} `json:"routes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Routes) == 0 {
		return nil, fmt.Errorf("no route found")
	}

	route := apiResp.Routes[0]
	durationSec := parseDuration(route.Duration)
	staticDurationSec := parseDuration(route.StaticDuration)

	now := time.Now()
	if req.DepartureTime != nil {
		now = *req.DepartureTime
	}

	result := &RouteResult{
		DistanceMeters:    route.DistanceMeters,
		DurationSeconds:   durationSec,
		StaticDuration:    staticDurationSec,
		TrafficDelay:      durationSec - staticDurationSec,
		EncodedPolyline:   route.Polyline.EncodedPolyline,
		DepartureTime:     now,
		ArrivalTime:       now.Add(time.Duration(durationSec) * time.Second),
	}

	if preference != RoutingPreferenceTrafficUnaware {
		result.DurationInTraffic = durationSec
	}

	for _, leg := range route.Legs {
		result.Legs = append(result.Legs, RouteLeg{
			DistanceMeters:  leg.DistanceMeters,
			DurationSeconds: parseDuration(leg.Duration),
			StartLocation: geo.Point{
				Lat: leg.StartLocation.LatLng.Latitude,
				Lng: leg.StartLocation.LatLng.Longitude,
			},
			EndLocation: geo.Point{
				Lat: leg.EndLocation.LatLng.Latitude,
				Lng: leg.EndLocation.LatLng.Longitude,
			},
		})
	}

	c.logger.Debug("route computed",
		"distance_m", result.DistanceMeters,
		"duration_s", result.DurationSeconds,
		"traffic_delay_s", result.TrafficDelay)

	return result, nil
}

// === Route Matrix ===

// ComputeRouteMatrixRequest represents a route matrix request.
type ComputeRouteMatrixRequest struct {
	Origins           []geo.Point
	Destinations      []geo.Point
	TravelMode        TravelMode
	RoutingPreference RoutingPreference
	DepartureTime     *time.Time
}

// RouteMatrixElement represents a single element in the route matrix.
type RouteMatrixElement struct {
	OriginIndex      int  `json:"origin_index"`
	DestinationIndex int  `json:"destination_index"`
	DistanceMeters   int  `json:"distance_meters"`
	DurationSeconds  int  `json:"duration_seconds"`
	Status           string `json:"status"` // "OK", "NOT_FOUND", etc.
}

// ComputeRouteMatrix computes a distance matrix between origins and destinations.
// Used for driver matching - find nearest drivers to pickup location.
func (c *Client) ComputeRouteMatrix(ctx context.Context, req *ComputeRouteMatrixRequest) ([]RouteMatrixElement, error) {
	ctx, span := c.startSpan(ctx, "maps.ComputeRouteMatrix")
	defer span.End()

	// Rate limiting
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx, "maps:compute_matrix"); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Build origins
	origins := make([]map[string]interface{}, len(req.Origins))
	for i, o := range req.Origins {
		origins[i] = map[string]interface{}{
			"waypoint": map[string]interface{}{
				"location": map[string]interface{}{
					"latLng": map[string]interface{}{
						"latitude":  o.Lat,
						"longitude": o.Lng,
					},
				},
			},
		}
	}

	// Build destinations
	destinations := make([]map[string]interface{}, len(req.Destinations))
	for i, d := range req.Destinations {
		destinations[i] = map[string]interface{}{
			"waypoint": map[string]interface{}{
				"location": map[string]interface{}{
					"latLng": map[string]interface{}{
						"latitude":  d.Lat,
						"longitude": d.Lng,
					},
				},
			},
		}
	}

	body := map[string]interface{}{
		"origins":      origins,
		"destinations": destinations,
		"travelMode":   req.TravelMode,
	}

	// Routing preference
	preference := req.RoutingPreference
	if preference == "" {
		if c.config.EnableTrafficRouting {
			preference = RoutingPreferenceTrafficAware
		} else {
			preference = RoutingPreferenceTrafficUnaware
		}
	}
	body["routingPreference"] = preference

	// Departure time
	if req.DepartureTime != nil {
		body["departureTime"] = req.DepartureTime.Format(time.RFC3339)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", computeMatrixURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Goog-Api-Key", c.config.APIKey)
	httpReq.Header.Set("X-Goog-FieldMask", "originIndex,destinationIndex,duration,distanceMeters,status,condition")

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// The response is a stream of JSON objects (one per element)
	// Read the entire response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse as array of elements
	var elements []struct {
		OriginIndex      int    `json:"originIndex"`
		DestinationIndex int    `json:"destinationIndex"`
		Duration         string `json:"duration"`
		DistanceMeters   int    `json:"distanceMeters"`
		Status           string `json:"status"`
		Condition        string `json:"condition"`
	}

	if err := json.Unmarshal(respBody, &elements); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]RouteMatrixElement, 0, len(elements))
	for _, e := range elements {
		status := "OK"
		if e.Status != "" {
			status = e.Status
		}
		if e.Condition == "ROUTE_NOT_FOUND" {
			status = "NOT_FOUND"
		}

		results = append(results, RouteMatrixElement{
			OriginIndex:      e.OriginIndex,
			DestinationIndex: e.DestinationIndex,
			DistanceMeters:   e.DistanceMeters,
			DurationSeconds:  parseDuration(e.Duration),
			Status:           status,
		})
	}

	c.logger.Debug("route matrix computed",
		"origins", len(req.Origins),
		"destinations", len(req.Destinations),
		"elements", len(results))

	return results, nil
}

// === Reverse Geocode ===

// ReverseGeocodeResult represents a reverse geocode result.
type ReverseGeocodeResult struct {
	PlaceID          string    `json:"place_id"`
	FormattedAddress string    `json:"formatted_address"`
	Location         geo.Point `json:"location"`
	AddressComponents []AddressComponent `json:"address_components"`
	City             string    `json:"city"`
	State            string    `json:"state"`
	Country          string    `json:"country"`
	PostalCode       string    `json:"postal_code"`
}

// ReverseGeocode converts coordinates to an address.
func (c *Client) ReverseGeocode(ctx context.Context, location geo.Point) (*ReverseGeocodeResult, error) {
	ctx, span := c.startSpan(ctx, "maps.ReverseGeocode")
	defer span.End()

	// Check cache first (round to ~100m precision)
	cacheKey := fmt.Sprintf("revgeo:%.4f,%.4f", location.Lat, location.Lng)
	if c.cache != nil {
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			var result ReverseGeocodeResult
			if err := json.Unmarshal(cached, &result); err == nil {
				c.logger.Debug("reverse geocode cache hit", "lat", location.Lat, "lng", location.Lng)
				return &result, nil
			}
		}
	}

	// Rate limiting
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx, "maps:reverse_geocode"); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Build URL
	params := url.Values{}
	params.Set("latlng", fmt.Sprintf("%f,%f", location.Lat, location.Lng))
	params.Set("key", c.config.APIKey)
	params.Set("result_type", "street_address|premise|sublocality|locality")

	reqURL := fmt.Sprintf("%s?%s", geocodeURL, params.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status  string `json:"status"`
		Results []struct {
			PlaceID          string `json:"place_id"`
			FormattedAddress string `json:"formatted_address"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
			AddressComponents []struct {
				LongName  string   `json:"long_name"`
				ShortName string   `json:"short_name"`
				Types     []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Status != "OK" || len(apiResp.Results) == 0 {
		return nil, fmt.Errorf("no results found for coordinates")
	}

	r := apiResp.Results[0]
	result := &ReverseGeocodeResult{
		PlaceID:          r.PlaceID,
		FormattedAddress: r.FormattedAddress,
		Location: geo.Point{
			Lat: r.Geometry.Location.Lat,
			Lng: r.Geometry.Location.Lng,
		},
	}

	// Parse address components
	for _, comp := range r.AddressComponents {
		result.AddressComponents = append(result.AddressComponents, AddressComponent{
			LongName:  comp.LongName,
			ShortName: comp.ShortName,
			Types:     comp.Types,
		})

		// Extract specific components
		for _, t := range comp.Types {
			switch t {
			case "locality":
				result.City = comp.LongName
			case "administrative_area_level_1":
				result.State = comp.ShortName
			case "country":
				result.Country = comp.ShortName
			case "postal_code":
				result.PostalCode = comp.LongName
			}
		}
	}

	// Cache the result
	if c.cache != nil {
		if cached, err := json.Marshal(result); err == nil {
			_ = c.cache.Set(ctx, cacheKey, cached, c.config.CacheTTL)
		}
	}

	c.logger.Debug("reverse geocode completed",
		"lat", location.Lat,
		"lng", location.Lng,
		"city", result.City)

	return result, nil
}

// === Helper Functions ===

// doRequest executes an HTTP request with retries.
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	var lastErr error

	for i := 0; i <= c.config.MaxRetries; i++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(c.config.RetryDelay * time.Duration(i+1))
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		// Read error body
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check for retryable errors
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("Google Maps API error: %d - %s", resp.StatusCode, string(body))
			time.Sleep(c.config.RetryDelay * time.Duration(i+1))
			continue
		}

		// Non-retryable error
		return nil, fmt.Errorf("Google Maps API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// startSpan starts a telemetry span if tracer is configured.
func (c *Client) startSpan(ctx context.Context, name string) (context.Context, *Span) {
	if c.tracer != nil {
		return c.tracer.StartSpan(ctx, name)
	}
	return ctx, &Span{}
}

// parseDuration parses a Google duration string (e.g., "123s") to seconds.
func parseDuration(d string) int {
	if d == "" {
		return 0
	}
	d = strings.TrimSuffix(d, "s")
	sec, _ := strconv.Atoi(d)
	return sec
}
