package test

import (
	"sync"
	"testing"
	"time"

	"github.com/osmike/fcache"
)

func TestReturnValuesAreCached(t *testing.T) {
	var mu sync.Mutex
	calls := 0

	fn := func(key int) (int, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return key * 2, nil
	}

	cache := fcache.NewCachedFunction(fn, &fcache.Config{
		TTL:      5 * time.Minute,
		Capacity: 100,
	}, &fcache.Hooks{})

	// First call: should invoke the underlying function
	v1, err := cache(5)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call: should return cached value instantly
	v2, err := cache(5)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if v1 != 10 || v2 != 10 {
		t.Errorf("expected both =10, got %d and %d", v1, v2)
	}

	// Underlying function should be called only once for the same key
	mu.Lock()
	if calls != 1 {
		t.Errorf("underlying called %d times; want 1", calls)
	}
	mu.Unlock()
}
