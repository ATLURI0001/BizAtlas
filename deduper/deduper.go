package deduper

import (
	"context"
	"sync"
)

type Deduper interface {
	AddIfNotExists(context.Context, string) bool
	Contains(context.Context, string) bool
	Delete(context.Context, string) bool
	Len() int
	Reset()
}

func New() Deduper {
	return &hashmap{
		seen: make(map[uint64]struct{}),
		mux:  &sync.RWMutex{},
	}
}
