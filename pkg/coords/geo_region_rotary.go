package coords

import "sync"

// GeoRegionRotary is a bounded ring buffer for geo region samples.
type GeoRegionRotary struct {
	mu      sync.Mutex
	buf     []string
	size    int
	head    int
	count   int
}

func NewGeoRegionRotary(size int) *GeoRegionRotary {
	return &GeoRegionRotary{buf: make([]string, size), size: size}
}

func (r *GeoRegionRotary) Push(region string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.head%r.size] = region
	r.head++
	if r.count < r.size {
		r.count++
	}
}

func (r *GeoRegionRotary) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}
