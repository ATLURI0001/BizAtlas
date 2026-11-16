package deduper

import (
	"context"
	"strings"
	"sync"
)

var _ Deduper = (*hashmap)(nil)

const (
	offset64 = 14695981039346656037
	prime64  = 1099511628211
)

type hashmap struct {
	mux  *sync.RWMutex
	seen map[uint64]struct{}
}

func (d *hashmap) AddIfNotExists(_ context.Context, key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	h := hash(key)

	d.mux.RLock()
	if _, ok := d.seen[h]; ok {
		d.mux.RUnlock()
		return false
	}
	d.mux.RUnlock()

	d.mux.Lock()
	defer d.mux.Unlock()

	if _, ok := d.seen[h]; ok {
		return false
	}

	d.seen[h] = struct{}{}

	return true
}

func (d *hashmap) Contains(_ context.Context, key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	h := hash(key)

	d.mux.RLock()
	defer d.mux.RUnlock()

	_, ok := d.seen[h]
	return ok
}

func (d *hashmap) Delete(_ context.Context, key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	h := hash(key)

	d.mux.Lock()
	defer d.mux.Unlock()

	if _, ok := d.seen[h]; ok {
		delete(d.seen, h)
		return true
	}

	return false
}

func (d *hashmap) AddBatch(_ context.Context, keys []string) int {
	if len(keys) == 0 {
		return 0
	}

	d.mux.Lock()
	defer d.mux.Unlock()

	added := 0
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		h := hash(key)
		if _, ok := d.seen[h]; !ok {
			d.seen[h] = struct{}{}
			added++
		}
	}
	return added
}

func (d *hashmap) Len() int {
	d.mux.RLock()
	defer d.mux.RUnlock()

	return len(d.seen)
}

func (d *hashmap) Reset() {
	d.mux.Lock()
	defer d.mux.Unlock()

	d.seen = make(map[uint64]struct{})
}

func hash(key string) uint64 {
	var h uint64 = offset64
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime64
	}
	return h
}
