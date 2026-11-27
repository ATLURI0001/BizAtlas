package coords

import "fmt"

// BoundingBoxContract validates geographic bounding boxes.
type BoundingBoxContract struct {
	MinLat, MaxLat float64
}

func NewBoundingBoxContract(minLat, maxLat float64) *BoundingBoxContract {
	return &BoundingBoxContract{MinLat: minLat, MaxLat: maxLat}
}

func (c *BoundingBoxContract) Validate(lat float64) error {
	if lat < c.MinLat || lat > c.MaxLat {
		return fmt.Errorf("latitude %.4f out of bounds [%.4f, %.4f]", lat, c.MinLat, c.MaxLat)
	}
	return nil
}
