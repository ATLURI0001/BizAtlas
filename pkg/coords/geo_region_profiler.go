package coords

// GeoRegionProfiler measures GeoRegion lookup latency samples in nanoseconds.
type GeoRegionProfiler struct {
	samples []int64
}

func (p *GeoRegionProfiler) Record(ns int64) {
	p.samples = append(p.samples, ns)
}

func (p *GeoRegionProfiler) Len() int { return len(p.samples) }
