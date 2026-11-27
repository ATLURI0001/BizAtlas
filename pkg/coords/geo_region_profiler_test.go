package coords

import "testing"

func TestGeoRegionProfiler(t *testing.T) {
	p := &GeoRegionProfiler{}
	p.Record(1200)
	p.Record(800)
	if p.Len() != 2 {
		t.Fatalf("expected 2 samples, got %d", p.Len())
	}
}
