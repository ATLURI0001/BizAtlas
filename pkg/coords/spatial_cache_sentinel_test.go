package coords

import (
	"testing"
	"time"
)

func TestSpatialCacheSentinel_CardinalityAndTTL(t *testing.T) {
	sentinel := NewSpatialCacheSentinel(2, 100*time.Millisecond)
	now := time.Now()

	if !sentinel.Put("k1", Coordinates{Latitude: 10, Longitude: 20}, now) {
		t.Fatal("expected k1 insert to succeed")
	}
	if !sentinel.Put("k2", Coordinates{Latitude: 30, Longitude: 40}, now) {
		t.Fatal("expected k2 insert to succeed")
	}

	// Capacity full -> k3 rejected
	if sentinel.Put("k3", Coordinates{Latitude: 50, Longitude: 60}, now) {
		t.Fatal("expected k3 insert to be rejected due to capacity cap")
	}

	// After TTL expires -> evict expired and allow k3
	later := now.Add(150 * time.Millisecond)
	if !sentinel.Put("k3", Coordinates{Latitude: 50, Longitude: 60}, later) {
		t.Fatal("expected k3 insert to succeed after TTL eviction")
	}

	_, evictions, hits := sentinel.Stats()
	if evictions != 2 || hits != 1 {
		t.Fatalf("expected 2 evictions and 1 incident hit, got %d, %d", evictions, hits)
	}
}
