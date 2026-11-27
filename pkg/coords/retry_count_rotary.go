package coords

// RetryCountRotary tracks bounded scrape retry counts.
type RetryCountRotary struct {
	samples []int
	cap     int
}

func NewRetryCountRotary(capacity int) *RetryCountRotary {
	return &RetryCountRotary{cap: capacity}
}

func (r *RetryCountRotary) Record(retries int) {
	if len(r.samples) >= r.cap {
		r.samples = r.samples[1:]
	}
	r.samples = append(r.samples, retries)
}

func (r *RetryCountRotary) Avg() float64 {
	if len(r.samples) == 0 {
		return 0
	}
	sum := 0
	for _, s := range r.samples {
		sum += s
	}
	return float64(sum) / float64(len(r.samples))
}
