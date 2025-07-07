package benchmark

import (
	"testing"

	"github.com/osmike/fcache"
)

func BenchmarkCachedCold(b *testing.B) {
	const delay = 10
	cached := fcache.NewCachedFunction(slowFunc, nil, nil) // default options: TTL=5m, LRU=1000
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Use a new key each time to simulate "cold" cache access (no hits)
		key := delay + i // unique key per iteration
		_, err := cached(key)
		if err != nil {
			b.Fatalf("err: %v", err)
		}
	}
}
