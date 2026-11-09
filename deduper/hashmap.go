package deduper

import (
	"context"
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

func hash(key string) uint64 {
	var h uint64 = offset64
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime64
	}
	return h
}
