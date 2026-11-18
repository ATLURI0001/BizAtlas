package deduper

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNormalizeShardCount(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, defaultShardCount},
		{-10, defaultShardCount},
		{1, 1},
		{3, 4},
		{16, 16},
		{17, 32},
		{5000, maxShardCount},
	}
	for _, tc := range cases {
		if got := normalizeShardCount(tc.in); got != tc.want {
			t.Fatalf("normalizeShardCount(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestShardedDeduperCoreOperationsAndContext(t *testing.T) {
	ctx := context.Background()
	d := NewSharded(16)

	if !d.AddIfNotExists(ctx, "  place-1  ") {
		t.Fatal("expected first insert to succeed")
	}
	if d.AddIfNotExists(ctx, "place-1") {
		t.Fatal("expected duplicate insert to return false")
	}
	if d.AddIfNotExists(ctx, "   ") {
		t.Fatal("expected whitespace-only key to be ignored")
	}
	if !d.Contains(ctx, "place-1") {
		t.Fatal("expected Contains to return true for inserted key")
	}
	if d.Len() != 1 {
		t.Fatalf("expected Len()=1, got %d", d.Len())
	}

	// Batch insert with duplicates and blanks
	added := d.AddBatch(ctx, []string{"place-1", "place-2", "place-3", "place-2", ""})
	if added != 2 {
		t.Fatalf("expected AddBatch to add 2 new keys, got %d", added)
	}
	if d.Len() != 3 {
		t.Fatalf("expected Len()=3, got %d", d.Len())
	}

	// Batch delete
	deleted := d.DeleteBatch(ctx, []string{"place-2", "nonexistent", "place-3", "place-2"})
	if deleted != 2 {
		t.Fatalf("expected DeleteBatch to remove 2 keys, got %d", deleted)
	}
	if d.Len() != 1 {
		t.Fatalf("expected Len()=1 after DeleteBatch, got %d", d.Len())
	}

	// Single delete
	if !d.Delete(ctx, "place-1") {
		t.Fatal("expected Delete to return true")
	}
	if d.Delete(ctx, "place-1") {
		t.Fatal("expected second Delete to return false")
	}
	if d.Len() != 0 {
		t.Fatalf("expected Len()=0, got %d", d.Len())
	}

	// Canceled context must short-circuit mutations
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	if d.AddIfNotExists(canceledCtx, "canceled-key") {
		t.Fatal("expected AddIfNotExists with canceled context to return false")
	}
	if d.AddBatch(canceledCtx, []string{"a", "b"}) != 0 {
		t.Fatal("expected AddBatch with canceled context to add 0 keys")
	}
	if d.Len() != 0 {
		t.Fatalf("expected Len()=0 after canceled ops, got %d", d.Len())
	}
}

func TestShardedDeduperConcurrentStressAndReset(t *testing.T) {
	ctx := context.Background()
	d := NewSharded(32)

	const workers = 16
	const keysPerWorker = 250
	var uniqueInserts atomic.Int64

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < keysPerWorker; i++ {
				// Intentional overlap across workers (keys 0..999) to force contention
				key := fmt.Sprintf("biz-%d", (id*37+i)%1000)
				if d.AddIfNotExists(ctx, key) {
					uniqueInserts.Add(1)
				}
				_ = d.Contains(ctx, key)
			}
		}(w)
	}
	wg.Wait()

	if got := int64(d.Len()); got != uniqueInserts.Load() {
		t.Fatalf("atomic Len()=%d diverged from unique inserts=%d", got, uniqueInserts.Load())
	}

	d.Reset()
	if d.Len() != 0 {
		t.Fatalf("expected Len()=0 after Reset(), got %d", d.Len())
	}
}

func BenchmarkShardedDeduperParallel(b *testing.B) {
	ctx := context.Background()
	d := NewSharded(64)
	var counter atomic.Uint64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := counter.Add(1)
			d.AddIfNotExists(ctx, fmt.Sprintf("entity-%d", n%8192))
		}
	})
}
