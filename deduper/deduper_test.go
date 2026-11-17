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

func TestDeduper_WhitespaceTrimmingAndEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d := deduper.New()

	require.False(t, d.AddIfNotExists(ctx, ""))
	require.False(t, d.AddIfNotExists(ctx, "   \t\n   "))
	require.Equal(t, 0, d.Len())

	require.True(t, d.AddIfNotExists(ctx, "  padded_key  "))
	require.True(t, d.Contains(ctx, "padded_key"))
	require.True(t, d.Contains(ctx, "  padded_key  "))
	require.False(t, d.AddIfNotExists(ctx, "padded_key"))
	require.Equal(t, 1, d.Len())

	d.Reset()
	require.Equal(t, 0, d.Len())
	require.False(t, d.Contains(ctx, "padded_key"))
	require.True(t, d.AddIfNotExists(ctx, "padded_key"))
}

func TestDeduper_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d := deduper.New()

	require.False(t, d.Delete(ctx, ""))
	require.False(t, d.Delete(ctx, "non_existent"))

	require.True(t, d.AddIfNotExists(ctx, "place_101"))
	require.Equal(t, 1, d.Len())
	require.True(t, d.Contains(ctx, "place_101"))

	require.True(t, d.Delete(ctx, "  place_101  "))
	require.Equal(t, 0, d.Len())
	require.False(t, d.Contains(ctx, "place_101"))

	require.False(t, d.Delete(ctx, "place_101"))
	require.True(t, d.AddIfNotExists(ctx, "place_101"))
}

func TestDeduper_AddBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d := deduper.New()

	require.Equal(t, 0, d.AddBatch(ctx, nil))
	require.Equal(t, 0, d.AddBatch(ctx, []string{"", "   "}))

	keys := []string{"key1", "key2", "  key1  ", "key3", ""}
	added := d.AddBatch(ctx, keys)
	require.Equal(t, 3, added)
	require.Equal(t, 3, d.Len())

	// Re-adding same keys should result in 0 added
	require.Equal(t, 0, d.AddBatch(ctx, []string{"key1", "key2", "key3"}))
	require.Equal(t, 3, d.Len())

	// Adding one new and one existing
	require.Equal(t, 1, d.AddBatch(ctx, []string{"key2", "key4"}))
	require.Equal(t, 4, d.Len())
	require.True(t, d.Contains(ctx, "key4"))
}

func TestDeduper_DeleteBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d := deduper.New()

	require.Equal(t, 0, d.DeleteBatch(ctx, nil))
	require.Equal(t, 0, d.DeleteBatch(ctx, []string{"", "   "}))

	keys := []string{"key1", "key2", "key3", "key4"}
	require.Equal(t, 4, d.AddBatch(ctx, keys))
	require.Equal(t, 4, d.Len())

	// Batch delete 2 existing keys and 1 whitespace / 1 missing key
	deleted := d.DeleteBatch(ctx, []string{"key1", "  key2  ", "missing_key", ""})
	require.Equal(t, 2, deleted)
	require.Equal(t, 2, d.Len())
	require.False(t, d.Contains(ctx, "key1"))
	require.False(t, d.Contains(ctx, "key2"))
	require.True(t, d.Contains(ctx, "key3"))
	require.True(t, d.Contains(ctx, "key4"))

	// Deleting already deleted keys returns 0
	require.Equal(t, 0, d.DeleteBatch(ctx, []string{"key1", "key2"}))
	require.Equal(t, 2, d.Len())
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
