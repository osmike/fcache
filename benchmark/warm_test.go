package benchmark

import (
	"testing"

	"github.com/osmike/fcache"
)

func BenchmarkCachedWarm(b *testing.B) {
	const delay = 10
	cached := fcache.NewCachedFunction(slowFunc, nil, nil)
	// Pre-warm the cache with a single entry
	_, _ = cached(delay)

	b.ReportAllocs()
	b.ResetTimer() // reset the timer to exclude setup time
	for i := 0; i < b.N; i++ {
		// Always use the same key to simulate warm (cache hit) access
		_, err := cached(delay)
		if err != nil {
			b.Fatalf("err: %v", err)
		}
	}
}
