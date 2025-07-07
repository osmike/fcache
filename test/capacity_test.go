package test

import (
	"sync"
	"testing"
	"time"

	"github.com/osmike/fcache"
)

func TestCacheCapacityLimitAndEviction(t *testing.T) {
	var mu sync.Mutex
	calls := 0

	fn := func(key int) (int, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return key, nil
	}

	// Use a small capacity to easily test eviction behavior
	cache := fcache.NewCachedFunction(fn, &fcache.Config{
		TTL:      5 * time.Minute,
		Capacity: 2,
	}, &fcache.Hooks{})

	// Fill the cache with keys 1 and 2
	cache(1) // call #1
	cache(2) // call #2

	// Access key 1 to make key 2 the least recently used
	cache(1)

	// Insert key 3, which should evict the oldest (key 2) due to capacity
	cache(3) // call #3

	// Now key 2 should be a cache miss and trigger a new call
	cache(2) // call #4

	mu.Lock()
	if calls != 4 {
		t.Errorf("underlying called %d times; want 4", calls)
	}
	mu.Unlock()
}
