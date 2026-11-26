package coords

import (
	"sync"
	"time"
)

// SpatialCacheSentinel monitors geohash cache cardinality and prevents OOM memory leaks
// under high-entropy coordinate query storms (INC-2026-BIZ-5512).
type SpatialCacheSentinel struct {
	mu           sync.RWMutex
	maxEntries   int
	ttl          time.Duration
	entries      map[string]cacheItem
	evictionLog  int
	incidentHits int
}

type cacheItem struct {
	coords    Coordinates
	expiresAt time.Time
}

// NewSpatialCacheSentinel creates a bounded TTL spatial cache sentinel.
func NewSpatialCacheSentinel(maxEntries int, ttl time.Duration) *SpatialCacheSentinel {
	if maxEntries <= 0 {
		maxEntries = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &SpatialCacheSentinel{
		maxEntries: maxEntries,
		ttl:        ttl,
		entries:    make(map[string]cacheItem),
	}
}

// Put stores a coordinate entry, enforcing cardinality cap and incident tracking.
func (s *SpatialCacheSentinel) Put(key string, coords Coordinates, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.entries) >= s.maxEntries {
		if _, exists := s.entries[key]; !exists {
			s.evictExpiredLocked(now)
			if len(s.entries) >= s.maxEntries {
				s.incidentHits++
				return false // Reject unbounded growth
			}
		}
	}

	s.entries[key] = cacheItem{
		coords:    coords,
		expiresAt: now.Add(s.ttl),
	}
	return true
}

// Get retrieves an entry if present and non-expired.
func (s *SpatialCacheSentinel) Get(key string, now time.Time) (Coordinates, bool) {
	s.mu.RLock()
	item, ok := s.entries[key]
	s.mu.RUnlock()

	if !ok || now.After(item.expiresAt) {
		return Coordinates{}, false
	}
	return item.coords, true
}

func (s *SpatialCacheSentinel) evictExpiredLocked(now time.Time) {
	for k, v := range s.entries {
		if now.After(v.expiresAt) {
			delete(s.entries, k)
			s.evictionLog++
		}
	}
}

// Stats returns cache cardinality and incident hit counts.
func (s *SpatialCacheSentinel) Stats() (int, int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries), s.evictionLog, s.incidentHits
}
