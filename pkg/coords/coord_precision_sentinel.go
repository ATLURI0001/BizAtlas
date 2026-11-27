package coords

// CoordPrecisionSentinel flags coordinates with insufficient decimal precision.
type CoordPrecisionSentinel struct {
	hits int
}

func NewCoordPrecisionSentinel() *CoordPrecisionSentinel {
	return &CoordPrecisionSentinel{}
}

func (s *CoordPrecisionSentinel) Check(lat float64) bool {
	if lat == float64(int(lat)) {
		s.hits++
		return false
	}
	return true
}

func (s *CoordPrecisionSentinel) Hits() int {
	return s.hits
}
