package cache_test

import (
	"testing"
	"time"

	"trace/internal/cache"
)

func TestTTLCache_SetGetExpires(t *testing.T) {
	c := cache.NewTTLCache[string, int](20 * time.Millisecond)
	c.Set("key", 42)

	value, ok := c.Get("key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}

	time.Sleep(30 * time.Millisecond)
	_, ok = c.Get("key")
	if ok {
		t.Fatal("expected cache entry to expire")
	}
}

func TestTTLCache_Purge(t *testing.T) {
	c := cache.NewTTLCache[string, string](time.Minute)
	c.Set("one", "1")
	c.Purge()
	if _, ok := c.Get("one"); ok {
		t.Fatal("expected cache to be empty after purge")
	}
}
