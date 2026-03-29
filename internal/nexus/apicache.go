package nexus

import (
	"sync"
	"time"
)

// apiCache stores JSON response bodies with a fixed TTL per entry.
type apiCache struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]cacheItem
}

type cacheItem struct {
	b     []byte
	until time.Time
}

func newAPICache(ttl time.Duration) *apiCache {
	return &apiCache{
		ttl: ttl,
		m:   make(map[string]cacheItem),
	}
}

func (a *apiCache) get(key string) ([]byte, bool) {
	if a == nil {
		return nil, false
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	it, ok := a.m[key]
	if !ok {
		return nil, false
	}
	if now.After(it.until) {
		delete(a.m, key)
		return nil, false
	}
	out := make([]byte, len(it.b))
	copy(out, it.b)
	return out, true
}

func (a *apiCache) set(key string, v []byte) {
	if a == nil {
		return
	}
	dup := make([]byte, len(v))
	copy(dup, v)
	a.mu.Lock()
	a.m[key] = cacheItem{b: dup, until: time.Now().Add(a.ttl)}
	a.mu.Unlock()
}
