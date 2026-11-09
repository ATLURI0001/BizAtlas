package deduper_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/gosom/google-maps-scraper/deduper"
	"github.com/stretchr/testify/require"
)

func TestDeduper_BasicOperations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d := deduper.New()

	require.True(t, d.AddIfNotExists(ctx, "apple"))
	require.False(t, d.AddIfNotExists(ctx, "apple"))

	require.True(t, d.AddIfNotExists(ctx, "banana"))
	require.False(t, d.AddIfNotExists(ctx, "banana"))

	require.False(t, d.AddIfNotExists(ctx, "apple"))
}

func TestDeduper_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d := deduper.New()

	const numGoroutines = 50
	const numKeys = 200

	var wg sync.WaitGroup
	var acceptedCounts [numKeys]int
	var mux sync.Mutex

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < numKeys; k++ {
				key := fmt.Sprintf("place_%d", k)
				if d.AddIfNotExists(ctx, key) {
					mux.Lock()
					acceptedCounts[k]++
					mux.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Invariant: each distinct key must be accepted exactly once across all concurrent workers
	for k := 0; k < numKeys; k++ {
		require.Equal(t, 1, acceptedCounts[k], fmt.Sprintf("key %d should be accepted exactly once", k))
	}
}

func BenchmarkDeduper_AddIfNotExists(b *testing.B) {
	ctx := context.Background()
	d := deduper.New()
	key := "https://maps.google.com/?cid=12345678901234567890"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		d.AddIfNotExists(ctx, key)
	}
}
