package test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/osmike/fcache"
)

func TestConcurrentDifferentMapKeys(t *testing.T) {
	// 1) Generate 4 large maps with 200 elements each
	makeMap := func(id int) map[string]int {
		m := make(map[string]int, 200)
		for j := 0; j < 200; j++ {
			m[fmt.Sprintf("key_%d_%d", id, j)] = j
		}
		return m
	}
	keys := []map[string]int{
		makeMap(0),
		makeMap(1),
		makeMap(2),
		makeMap(3),
	}

	var mu sync.Mutex
	calls := 0

	// Simulate an expensive source function
	fn := func(m map[string]int) (int, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return len(m), nil
	}

	cache := fcache.NewCachedFunction(fn, &fcache.Config{
		TTL:      time.Second,
		Capacity: 10,
	}, &fcache.Hooks{})

	// 2) Warm up: one call for each unique key
	for i, key := range keys {
		v, err := cache(key)
		if err != nil {
			t.Fatalf("warm-up for key %d returned error: %v", i, err)
		}
		if v != 200 {
			t.Fatalf("warm-up for key %d returned %d; want 200", i, v)
		}
	}

	// Ensure the underlying function was called exactly once per unique key
	mu.Lock()
	if calls != len(keys) {
		t.Fatalf("after warm-up: underlying function called %d times; want %d", calls, len(keys))
	}
	mu.Unlock()

	// 3) Concurrent cache hits: 5 goroutines per key
	const perKey = 5
	var wg sync.WaitGroup
	for _, key := range keys {
		for i := 0; i < perKey; i++ {
			wg.Add(1)
			go func(key map[string]int) {
				defer wg.Done()
				// On cache hit, the underlying function should not be called
				v, err := cache(key)
				if err != nil || v != 200 {
					t.Errorf("concurrent hit returned (%d, %v); want (200, nil)", v, err)
				}
			}(key)
		}
	}
	wg.Wait()

	// 4) Verify that no additional calls to the underlying function were made
	mu.Lock()
	if calls != len(keys) {
		t.Errorf("after concurrent hits: underlying function called %d times; want still %d", calls, len(keys))
	}
	mu.Unlock()
}
