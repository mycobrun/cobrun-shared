package cosmosdb

import (
	"testing"
	"time"
)

func TestGeoPoint(t *testing.T) {
	t.Run("NewGeoPoint", func(t *testing.T) {
		lat := 37.7749
		lng := -122.4194

		gp := NewGeoPoint(lat, lng)

		if gp.Type != "Point" {
			t.Errorf("expected Type=Point, got %s", gp.Type)
		}

		if len(gp.Coordinates) != 2 {
			t.Errorf("expected 2 coordinates, got %d", len(gp.Coordinates))
		}

		// GeoJSON format is [lng, lat]
		if gp.Coordinates[0] != lng {
			t.Errorf("expected first coordinate (lng)=%f, got %f", lng, gp.Coordinates[0])
		}

		if gp.Coordinates[1] != lat {
			t.Errorf("expected second coordinate (lat)=%f, got %f", lat, gp.Coordinates[1])
		}
	})

	t.Run("Lat and Lng methods", func(t *testing.T) {
		lat := 40.7128
		lng := -74.0060

		gp := NewGeoPoint(lat, lng)

		if gp.Lat() != lat {
			t.Errorf("expected Lat()=%f, got %f", lat, gp.Lat())
		}

		if gp.Lng() != lng {
			t.Errorf("expected Lng()=%f, got %f", lng, gp.Lng())
		}
	})

	t.Run("empty GeoPoint", func(t *testing.T) {
		gp := GeoPoint{}

		if gp.Lat() != 0 {
			t.Errorf("empty GeoPoint Lat() should return 0, got %f", gp.Lat())
		}

		if gp.Lng() != 0 {
			t.Errorf("empty GeoPoint Lng() should return 0, got %f", gp.Lng())
		}
	})
}

func TestTrip(t *testing.T) {
	now := time.Now()
	trip := Trip{
		ID:        "trip123",
		RequestID: "req123",
		RiderID:   "rider123",
		DriverID:  "driver123",
		VehicleID: "vehicle123",
		RideType:  "standard",
		Status:    "completed",
		PickupLocation: NewGeoPoint(37.7749, -122.4194),
		DropoffLocation: NewGeoPoint(37.7849, -122.4094),
		EstimatedDistanceMeters: 5000,
		EstimatedDurationSecs:   900,
		ActualDistanceMeters:    5200,
		ActualDurationSecs:      950,
		EstimatedFare:           25.00,
		ActualFare:              27.50,
		SurgeMultiplier:         1.2,
		Currency:                "USD",
		PaymentMethodID:         "pm123",
		PaymentStatus:           "completed",
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if trip.ID != "trip123" {
		t.Errorf("expected ID=trip123, got %s", trip.ID)
	}

	if trip.RiderID != "rider123" {
		t.Errorf("expected RiderID=rider123, got %s", trip.RiderID)
	}

	if trip.Status != "completed" {
		t.Errorf("expected Status=completed, got %s", trip.Status)
	}

	if trip.ActualFare != 27.50 {
		t.Errorf("expected ActualFare=27.50, got %f", trip.ActualFare)
	}

	if trip.PickupLocation.Lat() != 37.7749 {
		t.Errorf("expected PickupLocation.Lat=37.7749, got %f", trip.PickupLocation.Lat())
	}
}

func TestFareBreakdown(t *testing.T) {
	fare := FareBreakdown{
		BaseFare:      10.00,
		DistanceFare:  8.50,
		TimeFare:      5.00,
		SurgeFare:     3.00,
		ServiceFee:    2.50,
		PromoDiscount: 2.00,
		Tips:          5.00,
		Total:         32.00,
	}

	if fare.BaseFare != 10.00 {
		t.Errorf("expected BaseFare=10.00, got %f", fare.BaseFare)
	}

	expectedTotal := fare.BaseFare + fare.DistanceFare + fare.TimeFare +
		fare.SurgeFare + fare.ServiceFee - fare.PromoDiscount + fare.Tips

	if fare.Total != expectedTotal {
		t.Errorf("expected Total=%f, got %f", expectedTotal, fare.Total)
	}
}

func TestRideRequest(t *testing.T) {
	now := time.Now()
	req := RideRequest{
		ID:              "req123",
		RiderID:         "rider123",
		PickupLocation:  NewGeoPoint(37.7749, -122.4194),
		DropoffLocation: NewGeoPoint(37.7849, -122.4094),
		RideType:        "standard",
		Status:          "pending",
		EstimatedFare:   25.00,
		SurgeMultiplier: 1.0,
		Currency:        "USD",
		SearchAttempts:  0,
		CurrentRadius:   1000,
		CreatedAt:       now,
		UpdatedAt:       now,
		ExpiresAt:       now.Add(10 * time.Minute),
	}

	if req.ID != "req123" {
		t.Errorf("expected ID=req123, got %s", req.ID)
	}

	if req.RiderID != "rider123" {
		t.Errorf("expected RiderID=rider123, got %s", req.RiderID)
	}

	if req.Status != "pending" {
		t.Errorf("expected Status=pending, got %s", req.Status)
	}

	if req.ExpiresAt.Before(req.CreatedAt) {
		t.Error("ExpiresAt should be after CreatedAt")
	}
}

func TestDriverOffer(t *testing.T) {
	now := time.Now()
	offer := DriverOffer{
		ID:            "offer123",
		RequestID:     "req123",
		DriverID:      "driver123",
		Status:        "pending",
		PickupETASecs: 300,
		DistanceM:     2000,
		Fare:          25.00,
		ExpiresAt:     now.Add(15 * time.Second),
		CreatedAt:     now,
		TTL:           15,
	}

	if offer.ID != "offer123" {
		t.Errorf("expected ID=offer123, got %s", offer.ID)
	}

	if offer.DriverID != "driver123" {
		t.Errorf("expected DriverID=driver123, got %s", offer.DriverID)
	}

	if offer.TTL != 15 {
		t.Errorf("expected TTL=15, got %d", offer.TTL)
	}

	if offer.PickupETASecs != 300 {
		t.Errorf("expected PickupETASecs=300, got %d", offer.PickupETASecs)
	}
}

func TestDriverLocation(t *testing.T) {
	now := time.Now()
	loc := DriverLocation{
		ID:         "loc123",
		DriverID:   "driver123",
		Location:   NewGeoPoint(37.7749, -122.4194),
		Heading:    90.0,
		Speed:      25.5,
		Accuracy:   10.0,
		Status:     "available",
		City:       "sanfrancisco",
		Geohash:    "9q8yy",
		UpdatedAt:  now,
		ReceivedAt: now,
		TTL:        86400,
	}

	if loc.DriverID != "driver123" {
		t.Errorf("expected DriverID=driver123, got %s", loc.DriverID)
	}

	if loc.Status != "available" {
		t.Errorf("expected Status=available, got %s", loc.Status)
	}

	if loc.Location.Lat() != 37.7749 {
		t.Errorf("expected Location.Lat=37.7749, got %f", loc.Location.Lat())
	}

	if loc.Heading != 90.0 {
		t.Errorf("expected Heading=90.0, got %f", loc.Heading)
	}

	if loc.Speed != 25.5 {
		t.Errorf("expected Speed=25.5, got %f", loc.Speed)
	}

	if loc.City != "sanfrancisco" {
		t.Errorf("expected City=sanfrancisco, got %s", loc.City)
	}
}

func TestLocationHistory(t *testing.T) {
	now := time.Now()
	history := LocationHistory{
		ID:        "hist123",
		DriverID:  "driver123",
		Location:  NewGeoPoint(37.7749, -122.4194),
		Heading:   90.0,
		Speed:     25.5,
		TripID:    "trip123",
		Timestamp: now,
		TTL:       604800,
	}

	if history.DriverID != "driver123" {
		t.Errorf("expected DriverID=driver123, got %s", history.DriverID)
	}

	if history.TripID != "trip123" {
		t.Errorf("expected TripID=trip123, got %s", history.TripID)
	}

	if history.TTL != 604800 {
		t.Errorf("expected TTL=604800, got %d", history.TTL)
	}
}

func TestEvent(t *testing.T) {
	now := time.Now()
	event := Event{
		ID:         "event123",
		Type:       "trip.completed",
		EntityID:   "trip123",
		EntityType: "trip",
		Data: map[string]interface{}{
			"rider_id":  "rider123",
			"driver_id": "driver123",
			"fare":      27.50,
		},
		Metadata: &EventMetadata{
			Source:      "trip-service",
			Version:     "1.0",
			CorrelationID: "corr123",
		},
		Timestamp: now,
		TTL:       2592000,
	}

	if event.Type != "trip.completed" {
		t.Errorf("expected Type=trip.completed, got %s", event.Type)
	}

	if event.EntityType != "trip" {
		t.Errorf("expected EntityType=trip, got %s", event.EntityType)
	}

	if event.Data == nil {
		t.Error("Data should not be nil")
	}

	if event.Metadata == nil {
		t.Error("Metadata should not be nil")
	}

	if event.Metadata.Source != "trip-service" {
		t.Errorf("expected Metadata.Source=trip-service, got %s", event.Metadata.Source)
	}

	if len(event.Data) != 3 {
		t.Errorf("expected 3 data fields, got %d", len(event.Data))
	}
}

func TestEventMetadata(t *testing.T) {
	metadata := EventMetadata{
		Source:      "test-service",
		Version:     "2.0",
		CorrelationID: "corr456",
	}

	if metadata.Source != "test-service" {
		t.Errorf("expected Source=test-service, got %s", metadata.Source)
	}

	if metadata.Version != "2.0" {
		t.Errorf("expected Version=2.0, got %s", metadata.Version)
	}

	if metadata.CorrelationID != "corr456" {
		t.Errorf("expected CorrelationID=corr456, got %s", metadata.CorrelationID)
	}
}

func TestGeoPointEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{name: "north pole", lat: 90, lng: 0},
		{name: "south pole", lat: -90, lng: 0},
		{name: "date line", lat: 0, lng: 180},
		{name: "antimeridian", lat: 0, lng: -180},
		{name: "equator prime meridian", lat: 0, lng: 0},
		{name: "san francisco", lat: 37.7749, lng: -122.4194},
		{name: "sydney", lat: -33.8688, lng: 151.2093},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := NewGeoPoint(tt.lat, tt.lng)

			if gp.Type != "Point" {
				t.Errorf("expected Type=Point, got %s", gp.Type)
			}

			if gp.Lat() != tt.lat {
				t.Errorf("expected Lat=%f, got %f", tt.lat, gp.Lat())
			}

			if gp.Lng() != tt.lng {
				t.Errorf("expected Lng=%f, got %f", tt.lng, gp.Lng())
			}
		})
	}
}

func TestTripStatusFlow(t *testing.T) {
	statuses := []string{
		"requested",
		"accepted",
		"driver_en_route",
		"driver_arrived",
		"in_progress",
		"completed",
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			trip := Trip{
				ID:      "trip123",
				RiderID: "rider123",
				Status:  status,
			}

			if trip.Status != status {
				t.Errorf("expected Status=%s, got %s", status, trip.Status)
			}
		})
	}
}

func TestTripWithOptionalFields(t *testing.T) {
	now := time.Now()
	trip := Trip{
		ID:              "trip123",
		RiderID:         "rider123",
		Status:          "completed",
		PickupLocation:  NewGeoPoint(37.7749, -122.4194),
		DropoffLocation: NewGeoPoint(37.7849, -122.4094),
		CreatedAt:       now,
		UpdatedAt:       now,
		StartedAt:       &now,
		CompletedAt:     &now,
	}

	if trip.StartedAt == nil {
		t.Error("StartedAt should not be nil when set")
	}

	if trip.CompletedAt == nil {
		t.Error("CompletedAt should not be nil when set")
	}

	if trip.CancelledAt != nil {
		t.Error("CancelledAt should be nil for completed trip")
	}
}

func TestRideRequestScheduling(t *testing.T) {
	now := time.Now()
	futureTime := now.Add(2 * time.Hour)

	scheduledReq := RideRequest{
		ID:          "req123",
		RiderID:     "rider123",
		IsScheduled: true,
		ScheduledAt: &futureTime,
		CreatedAt:   now,
	}

	if !scheduledReq.IsScheduled {
		t.Error("expected IsScheduled=true")
	}

	if scheduledReq.ScheduledAt == nil {
		t.Error("ScheduledAt should not be nil for scheduled ride")
	}

	if !scheduledReq.ScheduledAt.After(scheduledReq.CreatedAt) {
		t.Error("ScheduledAt should be after CreatedAt")
	}

	// Immediate request
	immediateReq := RideRequest{
		ID:          "req456",
		RiderID:     "rider456",
		IsScheduled: false,
		ScheduledAt: nil,
		CreatedAt:   now,
	}

	if immediateReq.IsScheduled {
		t.Error("expected IsScheduled=false")
	}

	if immediateReq.ScheduledAt != nil {
		t.Error("ScheduledAt should be nil for immediate ride")
	}
}

func TestDriverOfferExpiry(t *testing.T) {
	now := time.Now()
	offer := DriverOffer{
		ID:        "offer123",
		CreatedAt: now,
		ExpiresAt: now.Add(15 * time.Second),
		TTL:       15,
	}

	if !offer.ExpiresAt.After(offer.CreatedAt) {
		t.Error("ExpiresAt should be after CreatedAt")
	}

	duration := offer.ExpiresAt.Sub(offer.CreatedAt)
	if duration != 15*time.Second {
		t.Errorf("expected expiry duration=15s, got %v", duration)
	}

	if offer.TTL != 15 {
		t.Errorf("expected TTL=15, got %d", offer.TTL)
	}
}

func TestEventDataStructure(t *testing.T) {
	event := Event{
		ID:   "event123",
		Type: "trip.completed",
		Data: map[string]interface{}{
			"string_field": "value",
			"int_field":    42,
			"float_field":  3.14,
			"bool_field":   true,
			"nested": map[string]interface{}{
				"key": "value",
			},
		},
	}

	if event.Data == nil {
		t.Fatal("Data should not be nil")
	}

	if event.Data["string_field"] != "value" {
		t.Error("string field not preserved")
	}

	if event.Data["int_field"] != 42 {
		t.Error("int field not preserved")
	}

	if event.Data["bool_field"] != true {
		t.Error("bool field not preserved")
	}

	nested, ok := event.Data["nested"].(map[string]interface{})
	if !ok {
		t.Error("nested object not preserved")
	} else {
		if nested["key"] != "value" {
			t.Error("nested value not preserved")
		}
	}
}
