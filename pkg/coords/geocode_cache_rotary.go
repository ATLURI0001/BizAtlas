package coords

// GeocodeCacheRotary is a bounded ring buffer for geocode cache keys.
type GeocodeCacheRotary struct {
	buf  []string
	size int
	head int
	cnt  int
}

func NewGeocodeCacheRotary(size int) *GeocodeCacheRotary {
	return &GeocodeCacheRotary{buf: make([]string, size), size: size}
}

func (r *GeocodeCacheRotary) Push(key string) {
	r.buf[r.head%r.size] = key
	r.head++
	if r.cnt < r.size {
		r.cnt++
	}
}

func (r *GeocodeCacheRotary) Count() int {
	return r.cnt
}
