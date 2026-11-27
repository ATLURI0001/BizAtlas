package coords

// QuotaBurnRotary tracks bounded API quota burn samples.
type QuotaBurnRotary struct {
	samples []float64
	cap     int
}

func NewQuotaBurnRotary(capacity int) *QuotaBurnRotary {
	return &QuotaBurnRotary{cap: capacity}
}

func (r *QuotaBurnRotary) Record(rate float64) {
	if len(r.samples) >= r.cap {
		r.samples = r.samples[1:]
	}
	r.samples = append(r.samples, rate)
}

func (r *QuotaBurnRotary) Avg() float64 {
	if len(r.samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range r.samples {
		sum += s
	}
	return sum / float64(len(r.samples))
}
