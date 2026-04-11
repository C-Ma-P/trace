package cache

import (
	"sync"
	"time"
)

type TTLCache[K comparable, V any] struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[K]cacheItem[V]
}

type cacheItem[V any] struct {
	value   V
	expires time.Time
}

func NewTTLCache[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
	return &TTLCache[K, V]{
		ttl:   ttl,
		items: make(map[K]cacheItem[V]),
	}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().After(item.expires) {
		delete(c.items, key)
		var zero V
		return zero, false
	}
	return item.value, true
}

func (c *TTLCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem[V]{
		value:   value,
		expires: time.Now().Add(c.ttl),
	}
}

func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *TTLCache[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]cacheItem[V])
}
