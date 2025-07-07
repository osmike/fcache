package test

import (
	"sync"
	"testing"
	"time"

	"github.com/osmike/fcache"
)

func TestConcurrentCallsAreDeduplicated(t *testing.T) {
	var mu sync.Mutex
	calls := 0

	// Function that sleeps to simulate a long-running operation
	fn := func(key int) (int, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		return key * 3, nil
	}

	cache := fcache.NewCachedFunction(fn, &fcache.Config{
		TTL:      time.Second,
		Capacity: 100,
	}, &fcache.Hooks{})

	const n = 10
	var wg sync.WaitGroup
	results := make([]int, n)
	errs := make([]error, n)

	// Launch n concurrent goroutines with the same key
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := cache(4)
			results[i] = r
			errs[i] = err
		}(i)
	}
	wg.Wait()

	// All goroutines should receive the same result and no error
	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d error: %v", i, err)
		}
		if results[i] != 12 {
			t.Errorf("goroutine %d got %d; want 12", i, results[i])
		}
	}

	// Underlying function should be called only once due to deduplication
	mu.Lock()
	if calls != 1 {
		t.Errorf("underlying called %d times; want 1", calls)
	}
	mu.Unlock()
}
