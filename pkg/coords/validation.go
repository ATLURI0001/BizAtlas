package coords

import "fmt"

// Coordinates represents a geographic point with latitude and longitude.
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// IsValid checks if geographic coordinates fall within standard Earth bounds.
func (c Coordinates) IsValid() bool {
	return c.Latitude >= -90.0 && c.Latitude <= 90.0 && c.Longitude >= -180.0 && c.Longitude <= 180.0
}

// String formats coordinates as a human-readable string.
func (c Coordinates) String() string {
	return fmt.Sprintf("%.6f, %.6f", c.Latitude, c.Longitude)
}
