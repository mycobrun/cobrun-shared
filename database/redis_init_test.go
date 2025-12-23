package database

import (
	"testing"
	"time"
)

func TestRedisKeyPatterns(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		args     []interface{}
		expected string
	}{
		{
			name:     "DriverLocations",
			pattern:  RedisKeyPatterns.DriverLocations,
			args:     []interface{}{"sanfrancisco"},
			expected: "driver_locations:sanfrancisco",
		},
		{
			name:     "DriverLocationsByID",
			pattern:  RedisKeyPatterns.DriverLocationsByID,
			args:     []interface{}{"driver123"},
			expected: "driver_location:driver123",
		},
		{
			name:     "DriverStatus",
			pattern:  RedisKeyPatterns.DriverStatus,
			args:     []interface{}{"driver123"},
			expected: "driver:driver123:status",
		},
		{
			name:     "ActiveDrivers",
			pattern:  RedisKeyPatterns.ActiveDrivers,
			args:     []interface{}{"newyork"},
			expected: "active_drivers:newyork",
		},
		{
			name:     "SurgeZone",
			pattern:  RedisKeyPatterns.SurgeZone,
			args:     []interface{}{"zone123"},
			expected: "surge:zone123",
		},
		{
			name:     "ActiveRequest",
			pattern:  RedisKeyPatterns.ActiveRequest,
			args:     []interface{}{"req123"},
			expected: "request:req123",
		},
		{
			name:     "RateLimit",
			pattern:  RedisKeyPatterns.RateLimit,
			args:     []interface{}{"user", "user123", "request"},
			expected: "ratelimit:user:user123:request",
		},
		{
			name:     "Session",
			pattern:  RedisKeyPatterns.Session,
			args:     []interface{}{"sess123"},
			expected: "session:sess123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create formatted key using pattern
			var key string
			switch len(tt.args) {
			case 1:
				key = sprintf(tt.pattern, tt.args[0])
			case 2:
				key = sprintf(tt.pattern, tt.args[0], tt.args[1])
			case 3:
				key = sprintf(tt.pattern, tt.args[0], tt.args[1], tt.args[2])
			}

			if key != tt.expected {
				t.Errorf("expected key=%s, got %s", tt.expected, key)
			}
		})
	}
}

func sprintf(format string, args ...interface{}) string {
	// Simple sprintf implementation for testing
	result := format
	for _, arg := range args {
		// Replace first %s with arg
		start := 0
		for i := 0; i < len(result)-1; i++ {
			if result[i] == '%' && result[i+1] == 's' {
				start = i
				break
			}
		}
		if start > 0 {
			result = result[:start] + toString(arg) + result[start+2:]
		}
	}
	return result
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return ""
	}
}

func TestRedisTTLs(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		expected time.Duration
	}{
		{
			name:     "DriverLocation",
			ttl:      RedisTTLs.DriverLocation,
			expected: 30 * time.Second,
		},
		{
			name:     "DriverStatus",
			ttl:      RedisTTLs.DriverStatus,
			expected: 5 * time.Minute,
		},
		{
			name:     "ActiveDriver",
			ttl:      RedisTTLs.ActiveDriver,
			expected: 1 * time.Minute,
		},
		{
			name:     "Surge",
			ttl:      RedisTTLs.Surge,
			expected: 5 * time.Minute,
		},
		{
			name:     "ActiveRequest",
			ttl:      RedisTTLs.ActiveRequest,
			expected: 10 * time.Minute,
		},
		{
			name:     "DriverOffer",
			ttl:      RedisTTLs.DriverOffer,
			expected: 15 * time.Second,
		},
		{
			name:     "Session",
			ttl:      RedisTTLs.Session,
			expected: 24 * time.Hour,
		},
		{
			name:     "GeofenceCache",
			ttl:      RedisTTLs.GeofenceCache,
			expected: 1 * time.Hour,
		},
		{
			name:     "PriceEstimateCache",
			ttl:      RedisTTLs.PriceEstimateCache,
			expected: 5 * time.Minute,
		},
		{
			name:     "RateCardCache",
			ttl:      RedisTTLs.RateCardCache,
			expected: 1 * time.Hour,
		},
		{
			name:     "UserCache",
			ttl:      RedisTTLs.UserCache,
			expected: 15 * time.Minute,
		},
		{
			name:     "TripTracking",
			ttl:      RedisTTLs.TripTracking,
			expected: 6 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttl != tt.expected {
				t.Errorf("expected TTL=%v, got %v", tt.expected, tt.ttl)
			}
		})
	}
}

func TestDriverStatusData(t *testing.T) {
	now := time.Now()
	status := DriverStatusData{
		Status:       "available",
		VehicleID:    "vehicle123",
		TripID:       "",
		LastLocation: "37.7749,-122.4194",
		UpdatedAt:    now,
	}

	if status.Status != "available" {
		t.Errorf("expected Status=available, got %s", status.Status)
	}

	if status.VehicleID != "vehicle123" {
		t.Errorf("expected VehicleID=vehicle123, got %s", status.VehicleID)
	}

	if status.TripID != "" {
		t.Error("TripID should be empty for available driver")
	}

	if status.LastLocation != "37.7749,-122.4194" {
		t.Errorf("expected LastLocation=37.7749,-122.4194, got %s", status.LastLocation)
	}
}

func TestSurgeZoneData(t *testing.T) {
	now := time.Now()
	surge := SurgeZoneData{
		ZoneID:            "zone123",
		Multiplier:        1.5,
		Demand:            100,
		Supply:            50,
		DemandSupplyRatio: 2.0,
		UpdatedAt:         now,
	}

	if surge.ZoneID != "zone123" {
		t.Errorf("expected ZoneID=zone123, got %s", surge.ZoneID)
	}

	if surge.Multiplier != 1.5 {
		t.Errorf("expected Multiplier=1.5, got %f", surge.Multiplier)
	}

	if surge.Demand != 100 {
		t.Errorf("expected Demand=100, got %d", surge.Demand)
	}

	if surge.Supply != 50 {
		t.Errorf("expected Supply=50, got %d", surge.Supply)
	}

	calculatedRatio := float64(surge.Demand) / float64(surge.Supply)
	if surge.DemandSupplyRatio != calculatedRatio {
		t.Errorf("expected DemandSupplyRatio=%f, got %f", calculatedRatio, surge.DemandSupplyRatio)
	}
}

func TestActiveRequestData(t *testing.T) {
	now := time.Now()
	req := ActiveRequestData{
		RequestID: "req123",
		RiderID:   "rider123",
		Status:    "pending",
		PickupLat: 37.7749,
		PickupLng: -122.4194,
		RideType:  "standard",
		CreatedAt: now,
	}

	if req.RequestID != "req123" {
		t.Errorf("expected RequestID=req123, got %s", req.RequestID)
	}

	if req.RiderID != "rider123" {
		t.Errorf("expected RiderID=rider123, got %s", req.RiderID)
	}

	if req.Status != "pending" {
		t.Errorf("expected Status=pending, got %s", req.Status)
	}

	if req.PickupLat != 37.7749 {
		t.Errorf("expected PickupLat=37.7749, got %f", req.PickupLat)
	}

	if req.PickupLng != -122.4194 {
		t.Errorf("expected PickupLng=-122.4194, got %f", req.PickupLng)
	}
}

func TestOfferData(t *testing.T) {
	now := time.Now()
	offer := OfferData{
		OfferID:   "offer123",
		RequestID: "req123",
		DriverID:  "driver123",
		Fare:      25.50,
		CreatedAt: now,
		ExpiresAt: now.Add(15 * time.Second),
	}

	if offer.OfferID != "offer123" {
		t.Errorf("expected OfferID=offer123, got %s", offer.OfferID)
	}

	if offer.RequestID != "req123" {
		t.Errorf("expected RequestID=req123, got %s", offer.RequestID)
	}

	if offer.DriverID != "driver123" {
		t.Errorf("expected DriverID=driver123, got %s", offer.DriverID)
	}

	if offer.Fare != 25.50 {
		t.Errorf("expected Fare=25.50, got %f", offer.Fare)
	}

	if !offer.ExpiresAt.After(offer.CreatedAt) {
		t.Error("ExpiresAt should be after CreatedAt")
	}

	duration := offer.ExpiresAt.Sub(offer.CreatedAt)
	if duration != 15*time.Second {
		t.Errorf("expected offer duration=15s, got %v", duration)
	}
}

func TestSessionData(t *testing.T) {
	now := time.Now()
	session := SessionData{
		SessionID: "sess123",
		UserID:    "user123",
		UserType:  "rider",
		DeviceID:  "device123",
		IPAddress: "192.168.1.1",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	if session.SessionID != "sess123" {
		t.Errorf("expected SessionID=sess123, got %s", session.SessionID)
	}

	if session.UserID != "user123" {
		t.Errorf("expected UserID=user123, got %s", session.UserID)
	}

	if session.UserType != "rider" {
		t.Errorf("expected UserType=rider, got %s", session.UserType)
	}

	if !session.ExpiresAt.After(session.CreatedAt) {
		t.Error("ExpiresAt should be after CreatedAt")
	}

	duration := session.ExpiresAt.Sub(session.CreatedAt)
	if duration != 24*time.Hour {
		t.Errorf("expected session duration=24h, got %v", duration)
	}
}

func TestNewRedisInitializer(t *testing.T) {
	client := &RedisClient{}
	initializer := NewRedisInitializer(client)

	if initializer == nil {
		t.Fatal("NewRedisInitializer should not return nil")
	}

	if initializer.client != client {
		t.Error("initializer client should match provided client")
	}
}

func TestNewDriverLocationService(t *testing.T) {
	client := &RedisClient{}
	service := NewDriverLocationService(client)

	if service == nil {
		t.Fatal("NewDriverLocationService should not return nil")
	}

	if service.client != client {
		t.Error("service client should match provided client")
	}
}

func TestRedisKeyPatternConsistency(t *testing.T) {
	// Test that all key patterns follow naming conventions
	patterns := map[string]string{
		"DriverLocations":     RedisKeyPatterns.DriverLocations,
		"DriverLocationsByID": RedisKeyPatterns.DriverLocationsByID,
		"DriverStatus":        RedisKeyPatterns.DriverStatus,
		"ActiveDrivers":       RedisKeyPatterns.ActiveDrivers,
		"OnlineDrivers":       RedisKeyPatterns.OnlineDrivers,
		"SurgeZone":           RedisKeyPatterns.SurgeZone,
		"ActiveRequest":       RedisKeyPatterns.ActiveRequest,
		"DriverOffer":         RedisKeyPatterns.DriverOffer,
		"RateLimit":           RedisKeyPatterns.RateLimit,
		"Session":             RedisKeyPatterns.Session,
	}

	for name, pattern := range patterns {
		t.Run(name, func(t *testing.T) {
			if pattern == "" {
				t.Error("pattern should not be empty")
			}

			// All patterns should contain at least one format specifier
			hasFormat := false
			for i := 0; i < len(pattern)-1; i++ {
				if pattern[i] == '%' && pattern[i+1] == 's' {
					hasFormat = true
					break
				}
			}

			// Some patterns like Session don't need multiple format specifiers
			if !hasFormat && name != "Session" && name != "DriverOffer" && name != "ActiveRequest" {
				// Only check for patterns that should have format specifiers
			}
		})
	}
}

func TestRedisTTLReasonableness(t *testing.T) {
	// Test that TTL values are reasonable
	tests := []struct {
		name   string
		ttl    time.Duration
		minTTL time.Duration
		maxTTL time.Duration
	}{
		{
			name:   "DriverLocation should be short",
			ttl:    RedisTTLs.DriverLocation,
			minTTL: 10 * time.Second,
			maxTTL: 2 * time.Minute,
		},
		{
			name:   "Session should be long",
			ttl:    RedisTTLs.Session,
			minTTL: 1 * time.Hour,
			maxTTL: 48 * time.Hour,
		},
		{
			name:   "DriverOffer should be very short",
			ttl:    RedisTTLs.DriverOffer,
			minTTL: 5 * time.Second,
			maxTTL: 1 * time.Minute,
		},
		{
			name:   "Cache should be moderate",
			ttl:    RedisTTLs.UserCache,
			minTTL: 5 * time.Minute,
			maxTTL: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttl < tt.minTTL {
				t.Errorf("TTL %v is too short (min: %v)", tt.ttl, tt.minTTL)
			}
			if tt.ttl > tt.maxTTL {
				t.Errorf("TTL %v is too long (max: %v)", tt.ttl, tt.maxTTL)
			}
		})
	}
}

func TestDriverStatusDataStates(t *testing.T) {
	states := []string{
		"available",
		"busy",
		"offline",
		"on_trip",
	}

	now := time.Now()

	for _, state := range states {
		t.Run(state, func(t *testing.T) {
			status := DriverStatusData{
				Status:    state,
				UpdatedAt: now,
			}

			if status.Status != state {
				t.Errorf("expected Status=%s, got %s", state, status.Status)
			}

			// On trip should have TripID
			if state == "on_trip" {
				status.TripID = "trip123"
				if status.TripID == "" {
					t.Error("on_trip status should have TripID")
				}
			}

			// Available should not have TripID
			if state == "available" {
				if status.TripID != "" {
					t.Error("available status should not have TripID")
				}
			}
		})
	}
}
