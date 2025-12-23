// Package geo provides geospatial utilities including H3 hexagonal indexing.
package geo

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/uber/h3-go/v4"
)

// H3Resolution defines the H3 resolution levels.
// Resolution 7: ~5.16 km² average hexagon area (~1.22 km edge)
// Resolution 8: ~0.74 km² average hexagon area (~0.46 km edge)
// Resolution 9: ~0.11 km² average hexagon area (~0.17 km edge)
type H3Resolution int

const (
	// H3ResolutionCity is for city-level operations (resolution 7)
	H3ResolutionCity H3Resolution = 7
	// H3ResolutionNeighborhood is for neighborhood-level operations (resolution 8)
	H3ResolutionNeighborhood H3Resolution = 8
	// H3ResolutionBlock is for block-level operations (resolution 9)
	H3ResolutionBlock H3Resolution = 9
)

// H3Index wraps H3 functionality for the Cobrun platform.
type H3Index struct {
	resolution int
}

// NewH3Index creates a new H3 indexer with the specified resolution.
func NewH3Index(resolution H3Resolution) *H3Index {
	return &H3Index{
		resolution: int(resolution),
	}
}

// LatLngToCell converts a lat/lng point to an H3 cell.
func (h *H3Index) LatLngToCell(p Point) h3.Cell {
	return h3.LatLngToCell(h3.LatLng{Lat: p.Lat, Lng: p.Lng}, h.resolution)
}

// CellToLatLng converts an H3 cell to its center point.
func (h *H3Index) CellToLatLng(cell h3.Cell) Point {
	latLng := h3.CellToLatLng(cell)
	return Point{Lat: latLng.Lat, Lng: latLng.Lng}
}

// CellToString converts an H3 cell to its string representation.
func (h *H3Index) CellToString(cell h3.Cell) string {
	return h3.IndexToString(uint64(cell))
}

// StringToCell converts a string to an H3 cell.
func (h *H3Index) StringToCell(s string) (h3.Cell, error) {
	index, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		// Try parsing as decimal
		index = h3.IndexFromString(s)
		if index == 0 {
			return 0, fmt.Errorf("invalid H3 cell string: %s", s)
		}
	}
	return h3.Cell(index), nil
}

// GetNeighbors returns all cells within k rings of the given cell.
// k=1 returns 7 cells (center + 6 neighbors)
// k=2 returns 19 cells
// k=3 returns 37 cells
func (h *H3Index) GetNeighbors(cell h3.Cell, kRings int) []h3.Cell {
	return h3.GridDisk(cell, kRings)
}

// GetRing returns only the cells in the kth ring around the center cell.
// Uses the difference between k-ring and (k-1)-ring.
func (h *H3Index) GetRing(cell h3.Cell, k int) []h3.Cell {
	if k == 0 {
		return []h3.Cell{cell}
	}

	// Get all cells up to k rings
	allCells := h3.GridDisk(cell, k)

	// Get all cells up to k-1 rings
	innerCells := make(map[h3.Cell]bool)
	for _, c := range h3.GridDisk(cell, k-1) {
		innerCells[c] = true
	}

	// Return only the outer ring
	var ring []h3.Cell
	for _, c := range allCells {
		if !innerCells[c] {
			ring = append(ring, c)
		}
	}
	return ring
}

// GetNeighborStrings returns neighbor cell strings for the given point.
func (h *H3Index) GetNeighborStrings(p Point, kRings int) []string {
	cell := h.LatLngToCell(p)
	neighbors := h.GetNeighbors(cell, kRings)

	result := make([]string, len(neighbors))
	for i, c := range neighbors {
		result[i] = c.String()
	}
	return result
}

// GetCellString returns the H3 cell string for a point.
func (h *H3Index) GetCellString(p Point) string {
	return h.LatLngToCell(p).String()
}

// H3Cell represents an H3 cell with metadata.
type H3Cell struct {
	Index      string  `json:"index"`
	Center     Point   `json:"center"`
	Resolution int     `json:"resolution"`
}

// GetCellInfo returns detailed information about the cell containing a point.
func (h *H3Index) GetCellInfo(p Point) H3Cell {
	cell := h.LatLngToCell(p)
	center := h.CellToLatLng(cell)

	return H3Cell{
		Index:      cell.String(),
		Center:     center,
		Resolution: h.resolution,
	}
}

// GridDistance calculates the grid distance between two cells.
func (h *H3Index) GridDistance(cell1, cell2 h3.Cell) int {
	return h3.GridDistance(cell1, cell2)
}

// GridPath returns the cells along the path between two cells.
func (h *H3Index) GridPath(cell1, cell2 h3.Cell) []h3.Cell {
	return h3.GridPath(cell1, cell2)
}

// CoverRadius returns H3 cells that cover a circular area.
func (h *H3Index) CoverRadius(center Point, radiusKm float64) []h3.Cell {
	// Calculate approximate number of rings needed
	// At resolution 7, each cell is about 1.22 km edge length
	// At resolution 8, each cell is about 0.46 km edge length
	// At resolution 9, each cell is about 0.17 km edge length

	edgeLength := h.getApproximateEdgeLength()
	kRings := int(radiusKm/edgeLength) + 1
	if kRings < 1 {
		kRings = 1
	}
	if kRings > 10 {
		kRings = 10 // Cap at reasonable limit
	}

	cell := h.LatLngToCell(center)
	return h.GetNeighbors(cell, kRings)
}

// getApproximateEdgeLength returns the approximate edge length in km for the resolution.
func (h *H3Index) getApproximateEdgeLength() float64 {
	switch h.resolution {
	case 7:
		return 1.22
	case 8:
		return 0.46
	case 9:
		return 0.17
	case 10:
		return 0.065
	default:
		return 1.0
	}
}

// H3DriverIndex provides efficient driver lookup using H3 cells.
type H3DriverIndex struct {
	h3Index   *H3Index
	cells     map[string][]string // cell -> driver IDs
	drivers   map[string]string   // driver ID -> cell
}

// NewH3DriverIndex creates a new driver index.
func NewH3DriverIndex(resolution H3Resolution) *H3DriverIndex {
	return &H3DriverIndex{
		h3Index:   NewH3Index(resolution),
		cells:     make(map[string][]string),
		drivers:   make(map[string]string),
	}
}

// UpdateDriver updates a driver's position in the index.
func (idx *H3DriverIndex) UpdateDriver(driverID string, location Point) {
	newCell := idx.h3Index.GetCellString(location)

	// Remove from old cell if exists
	if oldCell, exists := idx.drivers[driverID]; exists {
		if oldCell == newCell {
			return // No change
		}
		idx.removeFromCell(driverID, oldCell)
	}

	// Add to new cell
	idx.drivers[driverID] = newCell
	idx.cells[newCell] = append(idx.cells[newCell], driverID)
}

// RemoveDriver removes a driver from the index.
func (idx *H3DriverIndex) RemoveDriver(driverID string) {
	if cell, exists := idx.drivers[driverID]; exists {
		idx.removeFromCell(driverID, cell)
		delete(idx.drivers, driverID)
	}
}

// removeFromCell removes a driver from a specific cell.
func (idx *H3DriverIndex) removeFromCell(driverID, cell string) {
	drivers := idx.cells[cell]
	for i, id := range drivers {
		if id == driverID {
			idx.cells[cell] = append(drivers[:i], drivers[i+1:]...)
			break
		}
	}
	if len(idx.cells[cell]) == 0 {
		delete(idx.cells, cell)
	}
}

// GetDriversNearby returns driver IDs within k rings of the pickup location.
func (idx *H3DriverIndex) GetDriversNearby(pickup Point, kRings int) []string {
	cells := idx.h3Index.GetNeighborStrings(pickup, kRings)

	var drivers []string
	seen := make(map[string]bool)

	for _, cell := range cells {
		for _, driverID := range idx.cells[cell] {
			if !seen[driverID] {
				seen[driverID] = true
				drivers = append(drivers, driverID)
			}
		}
	}

	return drivers
}

// GetDriverCell returns the H3 cell for a driver.
func (idx *H3DriverIndex) GetDriverCell(driverID string) (string, bool) {
	cell, exists := idx.drivers[driverID]
	return cell, exists
}

// GetDriverCount returns the number of drivers in the index.
func (idx *H3DriverIndex) GetDriverCount() int {
	return len(idx.drivers)
}

// GetCellStats returns statistics about cell distribution.
func (idx *H3DriverIndex) GetCellStats() map[string]int {
	stats := make(map[string]int)
	for cell, drivers := range idx.cells {
		stats[cell] = len(drivers)
	}
	return stats
}

// H3HeatmapCell represents a cell in a demand/supply heatmap.
type H3HeatmapCell struct {
	Index         string  `json:"index"`
	Center        Point   `json:"center"`
	DriverCount   int     `json:"driver_count"`
	RequestCount  int     `json:"request_count"`
	DemandScore   float64 `json:"demand_score"`   // 0-1, higher = more demand
	SupplyScore   float64 `json:"supply_score"`   // 0-1, higher = more supply
	SurgeLevel    float64 `json:"surge_level"`    // Multiplier (1.0 = no surge)
	Color         string  `json:"color"`          // Hex color for visualization
}

// H3Heatmap manages demand/supply heatmaps using H3.
type H3Heatmap struct {
	h3Index *H3Index
	cells   map[string]*H3HeatmapCell
}

// NewH3Heatmap creates a new heatmap.
func NewH3Heatmap(resolution H3Resolution) *H3Heatmap {
	return &H3Heatmap{
		h3Index: NewH3Index(resolution),
		cells:   make(map[string]*H3HeatmapCell),
	}
}

// UpdateCell updates or creates a heatmap cell.
func (hm *H3Heatmap) UpdateCell(index string, driverCount, requestCount int) {
	center, err := hm.h3Index.StringToCell(index)
	if err != nil {
		return
	}

	cell := hm.cells[index]
	if cell == nil {
		cell = &H3HeatmapCell{
			Index:  index,
			Center: hm.h3Index.CellToLatLng(center),
		}
		hm.cells[index] = cell
	}

	cell.DriverCount = driverCount
	cell.RequestCount = requestCount
	cell.calculateScores()
}

// calculateScores calculates demand/supply scores and surge level.
func (c *H3HeatmapCell) calculateScores() {
	// Demand score: Based on request count (normalized)
	maxRequests := 50.0 // Expected max requests per cell
	c.DemandScore = min(float64(c.RequestCount)/maxRequests, 1.0)

	// Supply score: Based on driver count (normalized)
	maxDrivers := 20.0 // Expected max drivers per cell
	c.SupplyScore = min(float64(c.DriverCount)/maxDrivers, 1.0)

	// Surge level calculation
	if c.DriverCount == 0 && c.RequestCount > 0 {
		c.SurgeLevel = 2.0 // High surge when no drivers
	} else if c.DriverCount == 0 {
		c.SurgeLevel = 1.0
	} else {
		ratio := float64(c.RequestCount) / float64(c.DriverCount)
		if ratio > 3 {
			c.SurgeLevel = 2.0
		} else if ratio > 2 {
			c.SurgeLevel = 1.5
		} else if ratio > 1.5 {
			c.SurgeLevel = 1.25
		} else {
			c.SurgeLevel = 1.0
		}
	}

	// Color based on demand/supply ratio
	c.Color = c.calculateColor()
}

// calculateColor returns a hex color based on demand/supply ratio.
func (c *H3HeatmapCell) calculateColor() string {
	if c.DriverCount == 0 && c.RequestCount == 0 {
		return "#808080" // Gray - no activity
	}

	ratio := c.SurgeLevel

	switch {
	case ratio >= 2.0:
		return "#FF0000" // Red - very high demand
	case ratio >= 1.5:
		return "#FF6600" // Orange - high demand
	case ratio >= 1.25:
		return "#FFCC00" // Yellow - moderate demand
	case ratio >= 1.0:
		return "#00CC00" // Green - balanced
	default:
		return "#0066FF" // Blue - oversupply
	}
}

// GetCellsInRadius returns heatmap cells within a radius of a point.
func (hm *H3Heatmap) GetCellsInRadius(center Point, radiusKm float64) []*H3HeatmapCell {
	cells := hm.h3Index.CoverRadius(center, radiusKm)

	var result []*H3HeatmapCell
	for _, cell := range cells {
		index := cell.String()
		if hmc, exists := hm.cells[index]; exists {
			result = append(result, hmc)
		} else {
			// Return empty cell
			result = append(result, &H3HeatmapCell{
				Index:      index,
				Center:     hm.h3Index.CellToLatLng(cell),
				SurgeLevel: 1.0,
				Color:      "#808080",
			})
		}
	}

	return result
}

// DirectionFromBearing returns a cardinal direction from a bearing.
func DirectionFromBearing(bearing float64) string {
	directions := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	index := int((bearing + 22.5) / 45.0) % 8
	return directions[index]
}

// GetDirection returns the direction from point1 to point2.
func GetDirection(from, to Point) string {
	bearing := Bearing(from, to)
	return DirectionFromBearing(bearing)
}

// H3BatchMatcher provides batch matching optimization using H3.
type H3BatchMatcher struct {
	h3Index *H3Index
}

// NewH3BatchMatcher creates a new batch matcher.
func NewH3BatchMatcher(resolution H3Resolution) *H3BatchMatcher {
	return &H3BatchMatcher{
		h3Index: NewH3Index(resolution),
	}
}

// MatchPriority represents a driver-request match with priority score.
type MatchPriority struct {
	DriverID   string  `json:"driver_id"`
	RequestID  string  `json:"request_id"`
	Score      float64 `json:"score"`
	GridDist   int     `json:"grid_distance"`
	ETASeconds int     `json:"eta_seconds"`
}

// OptimizeBatch performs batch optimization for multiple requests and drivers.
// Returns optimal driver-request pairings.
func (bm *H3BatchMatcher) OptimizeBatch(
	requests []MatchPriority,
	drivers []MatchPriority,
) []MatchPriority {
	// Sort by score descending
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Score > requests[j].Score
	})

	// Greedy assignment (simplified Hungarian algorithm)
	assigned := make(map[string]bool) // Assigned drivers
	result := make([]MatchPriority, 0)

	for _, req := range requests {
		if assigned[req.DriverID] {
			continue
		}
		assigned[req.DriverID] = true
		result = append(result, req)
	}

	return result
}

// Helper function for min
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ValidateH3Cell validates an H3 cell string.
func ValidateH3Cell(cellStr string) error {
	index := h3.IndexFromString(cellStr)
	if index == 0 {
		return fmt.Errorf("invalid H3 cell: %s", cellStr)
	}
	return nil
}
