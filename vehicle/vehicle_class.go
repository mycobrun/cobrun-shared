// Package vehicle provides vehicle-related types and utilities.
package vehicle

// Class represents vehicle classification for ride matching.
type Class string

const (
	ClassStandard Class = "standard"  // Toyota Camry, Honda Accord, etc.
	ClassComfort  Class = "comfort"   // Newer cars with extra legroom
	ClassXL       Class = "xl"        // SUVs, minivans (6+ passengers)
	ClassPremium  Class = "premium"   // Luxury vehicles (BMW, Mercedes, etc.)
	ClassGreen    Class = "green"     // Electric/hybrid vehicles
)

// AllClasses returns all valid vehicle classes.
func AllClasses() []Class {
	return []Class{
		ClassStandard,
		ClassComfort,
		ClassXL,
		ClassPremium,
		ClassGreen,
	}
}

// IsValid checks if the vehicle class is valid.
func (c Class) IsValid() bool {
	switch c {
	case ClassStandard, ClassComfort, ClassXL, ClassPremium, ClassGreen:
		return true
	}
	return false
}

// String returns the string representation of the vehicle class.
func (c Class) String() string {
	return string(c)
}

// DisplayName returns a human-readable name for the vehicle class.
func (c Class) DisplayName() string {
	switch c {
	case ClassStandard:
		return "Standard"
	case ClassComfort:
		return "Comfort"
	case ClassXL:
		return "XL"
	case ClassPremium:
		return "Premium"
	case ClassGreen:
		return "Green"
	default:
		return "Unknown"
	}
}

// Description returns a description of the vehicle class.
func (c Class) Description() string {
	switch c {
	case ClassStandard:
		return "Standard sedan vehicles (Toyota Camry, Honda Accord, etc.)"
	case ClassComfort:
		return "Newer vehicles with extra legroom and premium features"
	case ClassXL:
		return "SUVs and minivans that seat up to 6 passengers"
	case ClassPremium:
		return "Luxury vehicles (BMW, Mercedes, Audi, Lexus)"
	case ClassGreen:
		return "Electric and hybrid eco-friendly vehicles"
	default:
		return "Unknown vehicle class"
	}
}

// MaxPassengers returns the maximum number of passengers for this vehicle class.
func (c Class) MaxPassengers() int {
	switch c {
	case ClassXL:
		return 6
	default:
		return 4
	}
}

// Hierarchy returns the hierarchy level (higher = more premium).
func (c Class) Hierarchy() int {
	switch c {
	case ClassStandard:
		return 1
	case ClassComfort:
		return 2
	case ClassGreen:
		return 2
	case ClassXL:
		return 3
	case ClassPremium:
		return 4
	default:
		return 0
	}
}

// CanFulfill checks if this vehicle class can fulfill a request for the target class.
// Premium vehicles can fulfill comfort requests, etc.
func (c Class) CanFulfill(target Class) bool {
	// Same class always works
	if c == target {
		return true
	}

	// Special cases for class compatibility
	switch target {
	case ClassStandard:
		// Any vehicle can fulfill standard rides
		return c.Hierarchy() >= ClassStandard.Hierarchy()
	case ClassComfort:
		// Comfort and Premium can fulfill comfort requests
		return c == ClassComfort || c == ClassPremium
	case ClassXL:
		// Only XL can fulfill XL (need the space)
		return false
	case ClassPremium:
		// Only premium can fulfill premium
		return false
	case ClassGreen:
		// Only green vehicles for green rides
		return false
	default:
		return false
	}
}

// Requirements represents requirements for a vehicle to qualify for a class.
type Requirements struct {
	Class             Class    `json:"class"`
	MinYear           int      `json:"min_year"`
	MaxAge            int      `json:"max_age"` // Maximum age in years
	RequiredFeatures  []string `json:"required_features,omitempty"`
	AllowedMakes      []string `json:"allowed_makes,omitempty"`
	AllowedModels     []string `json:"allowed_models,omitempty"`
	MinSeats          int      `json:"min_seats"`
	MinRating         float64  `json:"min_rating"` // Driver rating requirement
	RequireInspection bool     `json:"require_inspection"`
}

// ClassRequirements returns the requirements for each vehicle class.
var ClassRequirements = map[Class]Requirements{
	ClassStandard: {
		Class:             ClassStandard,
		MinYear:           2010,
		MaxAge:            15,
		MinSeats:          4,
		MinRating:         4.0,
		RequireInspection: true,
	},
	ClassComfort: {
		Class:            ClassComfort,
		MinYear:          2018,
		MaxAge:           7,
		MinSeats:         4,
		MinRating:        4.7,
		RequiredFeatures: []string{"leather_seats", "extra_legroom"},
		RequireInspection: true,
	},
	ClassXL: {
		Class:             ClassXL,
		MinYear:           2015,
		MaxAge:            10,
		MinSeats:          6,
		MinRating:         4.5,
		RequireInspection: true,
	},
	ClassPremium: {
		Class:   ClassPremium,
		MinYear: 2020,
		MaxAge:  5,
		MinSeats: 4,
		MinRating: 4.8,
		AllowedMakes: []string{
			"BMW", "Mercedes-Benz", "Audi", "Lexus", "Tesla",
			"Porsche", "Jaguar", "Cadillac", "Lincoln", "Genesis",
		},
		RequiredFeatures:  []string{"leather_seats", "premium_audio"},
		RequireInspection: true,
	},
	ClassGreen: {
		Class:            ClassGreen,
		MinYear:          2018,
		MaxAge:           7,
		MinSeats:         4,
		MinRating:        4.5,
		RequiredFeatures: []string{"electric", "hybrid"},
		RequireInspection: true,
	},
}
