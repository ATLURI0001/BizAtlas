package deduper

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
)

var _ Deduper = (*shardedHashmap)(nil)

const (
	defaultShardCount = 32
	maxShardCount     = 1024
)

type dedupShard struct {
	mux  sync.RWMutex
	seen map[uint64]struct{}
	// Pad to prevent false sharing across CPU cache lines (64 bytes).
	_ [40]byte
}

type shardedHashmap struct {
	shards []dedupShard
	mask   uint64
	count  atomic.Int64
}

func normalizeShardCount(requested int) int {
	if requested <= 0 {
		return defaultShardCount
	}
	if requested >= maxShardCount {
		return maxShardCount
	}
	n := 1
	for n < requested {
		n <<= 1
	}
	return n
}

func newShardedHashmap(shardCount int) *shardedHashmap {
	n := normalizeShardCount(shardCount)
	shards := make([]dedupShard, n)
	for i := range shards {
		shards[i].seen = make(map[uint64]struct{})
	}
	return &shardedHashmap{
		shards: shards,
		mask:   uint64(n - 1),
	}
}

func (d *shardedHashmap) shardFor(h uint64) *dedupShard {
	return &d.shards[h&d.mask]
}

func (d *shardedHashmap) ShardCount() int {
	return len(d.shards)
}

func (d *shardedHashmap) AddIfNotExists(ctx context.Context, key string) bool {
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	h := hash(key)
	shard := d.shardFor(h)

	shard.mux.RLock()
	if _, ok := shard.seen[h]; ok {
		shard.mux.RUnlock()
		return false
	}
	shard.mux.RUnlock()

	shard.mux.Lock()
	defer shard.mux.Unlock()

	if _, ok := shard.seen[h]; ok {
		return false
	}
	shard.seen[h] = struct{}{}
	d.count.Add(1)
	return true
}

func (d *shardedHashmap) Contains(ctx context.Context, key string) bool {
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	h := hash(key)
	shard := d.shardFor(h)

	shard.mux.RLock()
	defer shard.mux.RUnlock()

	_, ok := shard.seen[h]
	return ok
}

func (d *shardedHashmap) Delete(ctx context.Context, key string) bool {
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	h := hash(key)
	shard := d.shardFor(h)

	shard.mux.Lock()
	defer shard.mux.Unlock()

	if _, ok := shard.seen[h]; ok {
		delete(shard.seen, h)
		d.count.Add(-1)
		return true
	}
	return false
}

func (d *shardedHashmap) AddBatch(ctx context.Context, keys []string) int {
	if len(keys) == 0 || (ctx != nil && ctx.Err() != nil) {
		return 0
	}

	// Partition hashes by shard index so each touched shard acquires its write lock once.
	grouped := make(map[uint64][]uint64)
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		h := hash(key)
		idx := h & d.mask
		grouped[idx] = append(grouped[idx], h)
	}

	added := 0
	for idx, hashes := range grouped {
		if ctx != nil && ctx.Err() != nil {
			break
		}
		shard := &d.shards[idx]
		shard.mux.Lock()
		shardAdded := 0
		for _, h := range hashes {
			if _, ok := shard.seen[h]; !ok {
				shard.seen[h] = struct{}{}
				shardAdded++
			}
		}
		shard.mux.Unlock()
		if shardAdded > 0 {
			d.count.Add(int64(shardAdded))
			added += shardAdded
		}
	}
	return added
}

func (d *shardedHashmap) DeleteBatch(ctx context.Context, keys []string) int {
	if len(keys) == 0 || (ctx != nil && ctx.Err() != nil) {
		return 0
	}

	grouped := make(map[uint64][]uint64)
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		h := hash(key)
		idx := h & d.mask
		grouped[idx] = append(grouped[idx], h)
	}

	deleted := 0
	for idx, hashes := range grouped {
		if ctx != nil && ctx.Err() != nil {
			break
		}
		shard := &d.shards[idx]
		shard.mux.Lock()
		shardDeleted := 0
		for _, h := range hashes {
			if _, ok := shard.seen[h]; ok {
				delete(shard.seen, h)
				shardDeleted++
			}
		}
		shard.mux.Unlock()
		if shardDeleted > 0 {
			d.count.Add(-int64(shardDeleted))
			deleted += shardDeleted
		}
	}
	return deleted
}

func (d *shardedHashmap) Len() int {
	n := d.count.Load()
	if n < 0 {
		return 0
	}
	return int(n)
}

func (d *shardedHashmap) Reset() {
	for i := range d.shards {
		shard := &d.shards[i]
		shard.mux.Lock()
		cleared := len(shard.seen)
		shard.seen = make(map[uint64]struct{})
		shard.mux.Unlock()
		if cleared > 0 {
			d.count.Add(-int64(cleared))
		}
	}
}
