package coords

// ScrapeLatencyRotary is a bounded ring buffer for scrape latency samples in ms.
type ScrapeLatencyRotary struct {
	samples []float64
	cap     int
}

func NewScrapeLatencyRotary(capacity int) *ScrapeLatencyRotary {
	return &ScrapeLatencyRotary{cap: capacity}
}

func (r *ScrapeLatencyRotary) Record(ms float64) {
	if len(r.samples) >= r.cap {
		r.samples = r.samples[1:]
	}
	r.samples = append(r.samples, ms)
}

func (r *ScrapeLatencyRotary) Avg() float64 {
	if len(r.samples) == 0 {
		return 0
	}
	var sum float64
	for _, v := range r.samples {
		sum += v
	}
	return sum / float64(len(r.samples))
}
