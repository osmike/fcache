package test

import (
	"sync"
	"testing"
	"time"

	"github.com/osmike/fcache"
)

func TestResultsExpireAfterTTL(t *testing.T) {
	var mu sync.Mutex
	calls := 0

	fn := func(key int) (int, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return key + 1, nil
	}

	cache := fcache.NewCachedFunction(fn, &fcache.Config{
		TTL:      50 * time.Millisecond,
		Capacity: 100,
	}, &fcache.Hooks{})

	// First call: should invoke the underlying function
	if v, _ := cache(7); v != 8 {
		t.Fatal("unexpected value")
	}
	// Second call: should return cached value (not expired)
	if v, _ := cache(7); v != 8 {
		t.Fatal("unexpected value")
	}

	// Underlying function should be called only once before expiry
	mu.Lock()
	if calls != 1 {
		t.Errorf("calls before expiry = %d; want 1", calls)
	}
	mu.Unlock()

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// After expiry, should invoke the underlying function again
	if v, _ := cache(7); v != 8 {
		t.Fatal("unexpected value after expiry")
	}
	mu.Lock()
	if calls != 2 {
		t.Errorf("calls after expiry = %d; want 2", calls)
	}
	mu.Unlock()
}
