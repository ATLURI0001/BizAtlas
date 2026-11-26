package coords

import "fmt"

// PlaceFilterContract validates required fields in Place search results.
type PlaceFilterContract struct{}

// Place represents a minimal place search result.
type Place struct {
	Name      string
	Latitude  float64
	Longitude float64
	Rating    float64
}

// Validate checks required fields and coordinate bounds.
func (c *PlaceFilterContract) Validate(p Place) error {
	if p.Name == "" {
		return fmt.Errorf("contract violation: place name must not be empty")
	}
	if p.Latitude < -90 || p.Latitude > 90 {
		return fmt.Errorf("contract violation: latitude %v out of range [-90, 90]", p.Latitude)
	}
	if p.Longitude < -180 || p.Longitude > 180 {
		return fmt.Errorf("contract violation: longitude %v out of range [-180, 180]", p.Longitude)
	}
	if p.Rating < 0 || p.Rating > 5 {
		return fmt.Errorf("contract violation: rating %v out of range [0, 5]", p.Rating)
	}
	return nil
}
