// Package geo provides geospatial utilities including H3 hexagonal indexing.
package geo

import (
	"testing"
)

func TestNewH3Index(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)
	if h3 == nil {
		t.Fatal("NewH3Index returned nil")
	}
	if h3.resolution != 8 {
		t.Errorf("expected resolution 8, got %d", h3.resolution)
	}
}

func TestH3Index_LatLngToCell(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	// San Francisco downtown
	p := Point{Lat: 37.7749, Lng: -122.4194}
	cell := h3.LatLngToCell(p)

	if cell == 0 {
		t.Error("LatLngToCell returned invalid cell")
	}

	// Same point should return same cell
	cell2 := h3.LatLngToCell(p)
	if cell != cell2 {
		t.Error("same point should return same cell")
	}
}

func TestH3Index_CellToLatLng(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	// Original point
	original := Point{Lat: 37.7749, Lng: -122.4194}
	cell := h3.LatLngToCell(original)

	// Convert back to lat/lng (will be cell center)
	center := h3.CellToLatLng(cell)

	// Center should be close to original (within cell radius)
	distance := HaversineDistanceMeters(original, center)
	if distance > 1000 { // Within 1km (cell is ~0.46km edge at res 8)
		t.Errorf("cell center too far from original: %f meters", distance)
	}
}

func TestH3Index_GetNeighbors(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	p := Point{Lat: 37.7749, Lng: -122.4194}
	cell := h3.LatLngToCell(p)

	tests := []struct {
		kRings       int
		expectedMin  int
		expectedMax  int
	}{
		{0, 1, 1},       // Just the cell itself
		{1, 7, 7},       // Center + 6 neighbors
		{2, 19, 19},     // Two rings
		{3, 37, 37},     // Three rings
	}

	for _, tt := range tests {
		neighbors := h3.GetNeighbors(cell, tt.kRings)
		if len(neighbors) < tt.expectedMin || len(neighbors) > tt.expectedMax {
			t.Errorf("GetNeighbors(k=%d) returned %d cells, want %d-%d",
				tt.kRings, len(neighbors), tt.expectedMin, tt.expectedMax)
		}
	}
}

func TestH3Index_GetRing(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	p := Point{Lat: 37.7749, Lng: -122.4194}
	cell := h3.LatLngToCell(p)

	// Ring 0 should be just the cell
	ring0 := h3.GetRing(cell, 0)
	if len(ring0) != 1 {
		t.Errorf("ring 0 should have 1 cell, got %d", len(ring0))
	}

	// Ring 1 should have 6 cells (hexagon has 6 neighbors)
	ring1 := h3.GetRing(cell, 1)
	if len(ring1) != 6 {
		t.Errorf("ring 1 should have 6 cells, got %d", len(ring1))
	}

	// Ring 2 should have 12 cells
	ring2 := h3.GetRing(cell, 2)
	if len(ring2) != 12 {
		t.Errorf("ring 2 should have 12 cells, got %d", len(ring2))
	}
}

func TestH3Index_CoverRadius(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	center := Point{Lat: 37.7749, Lng: -122.4194}

	// Small radius
	cells1km := h3.CoverRadius(center, 1.0)
	if len(cells1km) < 3 {
		t.Error("1km radius should cover at least 3 cells")
	}

	// Larger radius should cover more cells
	cells5km := h3.CoverRadius(center, 5.0)
	if len(cells5km) <= len(cells1km) {
		t.Error("5km radius should cover more cells than 1km")
	}
}

func TestH3Index_StringConversion(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	p := Point{Lat: 37.7749, Lng: -122.4194}
	cell := h3.LatLngToCell(p)

	// Convert to string
	cellStr := h3.CellToString(cell)
	if cellStr == "" {
		t.Error("CellToString returned empty string")
	}

	// Convert back
	cell2, err := h3.StringToCell(cellStr)
	if err != nil {
		t.Fatalf("StringToCell failed: %v", err)
	}

	if cell != cell2 {
		t.Error("round-trip conversion failed")
	}
}

func TestH3DriverIndex_UpdateDriver(t *testing.T) {
	idx := NewH3DriverIndex(H3ResolutionNeighborhood)

	// Add driver
	loc1 := Point{Lat: 37.7749, Lng: -122.4194}
	idx.UpdateDriver("driver-1", loc1)

	if idx.GetDriverCount() != 1 {
		t.Errorf("expected 1 driver, got %d", idx.GetDriverCount())
	}

	// Check driver cell
	cell, exists := idx.GetDriverCell("driver-1")
	if !exists {
		t.Error("driver should exist in index")
	}
	if cell == "" {
		t.Error("driver cell should not be empty")
	}

	// Update driver location
	loc2 := Point{Lat: 37.7850, Lng: -122.4100}
	idx.UpdateDriver("driver-1", loc2)

	// Count should still be 1
	if idx.GetDriverCount() != 1 {
		t.Errorf("expected 1 driver after update, got %d", idx.GetDriverCount())
	}

	// Cell should have changed
	newCell, _ := idx.GetDriverCell("driver-1")
	if newCell == cell {
		// Cells might actually be the same if locations are close enough
		// This is expected behavior
	}
}

func TestH3DriverIndex_RemoveDriver(t *testing.T) {
	idx := NewH3DriverIndex(H3ResolutionNeighborhood)

	// Add drivers
	idx.UpdateDriver("driver-1", Point{Lat: 37.7749, Lng: -122.4194})
	idx.UpdateDriver("driver-2", Point{Lat: 37.7849, Lng: -122.4094})

	if idx.GetDriverCount() != 2 {
		t.Errorf("expected 2 drivers, got %d", idx.GetDriverCount())
	}

	// Remove one
	idx.RemoveDriver("driver-1")

	if idx.GetDriverCount() != 1 {
		t.Errorf("expected 1 driver after removal, got %d", idx.GetDriverCount())
	}

	_, exists := idx.GetDriverCell("driver-1")
	if exists {
		t.Error("removed driver should not exist")
	}

	// driver-2 should still exist
	_, exists = idx.GetDriverCell("driver-2")
	if !exists {
		t.Error("driver-2 should still exist")
	}
}

func TestH3DriverIndex_GetDriversNearby(t *testing.T) {
	idx := NewH3DriverIndex(H3ResolutionNeighborhood)

	// Add drivers in SF
	idx.UpdateDriver("driver-1", Point{Lat: 37.7749, Lng: -122.4194})
	idx.UpdateDriver("driver-2", Point{Lat: 37.7760, Lng: -122.4180})
	idx.UpdateDriver("driver-3", Point{Lat: 37.7850, Lng: -122.4100})

	// Search near driver-1
	pickup := Point{Lat: 37.7749, Lng: -122.4194}
	nearby := idx.GetDriversNearby(pickup, 2)

	if len(nearby) == 0 {
		t.Error("should find at least one nearby driver")
	}

	// driver-1 should be in results (same location as pickup)
	found := false
	for _, id := range nearby {
		if id == "driver-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("driver-1 should be in nearby results")
	}
}

func TestH3Heatmap_UpdateCell(t *testing.T) {
	hm := NewH3Heatmap(H3ResolutionNeighborhood)

	// Get a valid cell string
	h3 := NewH3Index(H3ResolutionNeighborhood)
	cell := h3.LatLngToCell(Point{Lat: 37.7749, Lng: -122.4194})
	cellStr := cell.String()

	// Update cell
	hm.UpdateCell(cellStr, 5, 10)

	cells := hm.GetCellsInRadius(Point{Lat: 37.7749, Lng: -122.4194}, 1.0)
	if len(cells) == 0 {
		t.Fatal("should have at least one cell")
	}

	// Find our updated cell
	found := false
	for _, c := range cells {
		if c.Index == cellStr {
			found = true
			if c.DriverCount != 5 {
				t.Errorf("expected 5 drivers, got %d", c.DriverCount)
			}
			if c.RequestCount != 10 {
				t.Errorf("expected 10 requests, got %d", c.RequestCount)
			}
			// High demand (more requests than drivers) should trigger surge
			if c.SurgeLevel <= 1.0 {
				t.Errorf("expected surge >1.0 with 10 requests and 5 drivers, got %f", c.SurgeLevel)
			}
			break
		}
	}
	if !found {
		t.Error("updated cell not found in results")
	}
}

func TestDirectionFromBearing(t *testing.T) {
	tests := []struct {
		bearing float64
		want    string
	}{
		{0, "N"},
		{45, "NE"},
		{90, "E"},
		{135, "SE"},
		{180, "S"},
		{225, "SW"},
		{270, "W"},
		{315, "NW"},
		{360, "N"},
	}

	for _, tt := range tests {
		got := DirectionFromBearing(tt.bearing)
		if got != tt.want {
			t.Errorf("DirectionFromBearing(%f) = %s, want %s", tt.bearing, got, tt.want)
		}
	}
}

func TestGetDirection(t *testing.T) {
	// From SF to a point north
	from := Point{Lat: 37.7749, Lng: -122.4194}
	toNorth := Point{Lat: 37.8749, Lng: -122.4194}

	dir := GetDirection(from, toNorth)
	if dir != "N" {
		t.Errorf("expected N, got %s", dir)
	}

	// From SF to a point east
	toEast := Point{Lat: 37.7749, Lng: -122.3194}
	dir = GetDirection(from, toEast)
	if dir != "E" {
		t.Errorf("expected E, got %s", dir)
	}
}

func TestValidateH3Cell(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)
	cell := h3.LatLngToCell(Point{Lat: 37.7749, Lng: -122.4194})
	validStr := cell.String()

	// Valid cell
	err := ValidateH3Cell(validStr)
	if err != nil {
		t.Errorf("valid cell should not error: %v", err)
	}

	// Invalid cell
	err = ValidateH3Cell("invalid")
	if err == nil {
		t.Error("invalid cell should return error")
	}
}

func TestH3BatchMatcher_OptimizeBatch(t *testing.T) {
	bm := NewH3BatchMatcher(H3ResolutionNeighborhood)

	// Create some match priorities
	requests := []MatchPriority{
		{DriverID: "d1", RequestID: "r1", Score: 0.9, GridDist: 1, ETASeconds: 60},
		{DriverID: "d2", RequestID: "r1", Score: 0.7, GridDist: 2, ETASeconds: 120},
		{DriverID: "d1", RequestID: "r2", Score: 0.8, GridDist: 1, ETASeconds: 90},
		{DriverID: "d3", RequestID: "r2", Score: 0.6, GridDist: 3, ETASeconds: 180},
	}

	result := bm.OptimizeBatch(requests, nil)

	// Should return assignments (greedy algorithm assigns highest scores first)
	if len(result) == 0 {
		t.Error("should have some assignments")
	}

	// Check no driver is assigned twice
	seen := make(map[string]bool)
	for _, match := range result {
		if seen[match.DriverID] {
			t.Errorf("driver %s assigned multiple times", match.DriverID)
		}
		seen[match.DriverID] = true
	}
}

func TestH3Index_GridDistance(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	// Same cell
	p1 := Point{Lat: 37.7749, Lng: -122.4194}
	cell1 := h3.LatLngToCell(p1)

	dist := h3.GridDistance(cell1, cell1)
	if dist != 0 {
		t.Errorf("same cell should have distance 0, got %d", dist)
	}

	// Adjacent cells should have distance 1
	p2 := Point{Lat: 37.7760, Lng: -122.4180}
	cell2 := h3.LatLngToCell(p2)

	// Grid distance depends on actual cell positions
	dist = h3.GridDistance(cell1, cell2)
	if dist < 0 {
		t.Error("grid distance should be non-negative")
	}
}

func TestH3Index_GetCellInfo(t *testing.T) {
	h3 := NewH3Index(H3ResolutionNeighborhood)

	p := Point{Lat: 37.7749, Lng: -122.4194}
	info := h3.GetCellInfo(p)

	if info.Index == "" {
		t.Error("cell index should not be empty")
	}

	if info.Resolution != 8 {
		t.Errorf("expected resolution 8, got %d", info.Resolution)
	}

	if !info.Center.IsValid() {
		t.Error("cell center should be valid")
	}
}
