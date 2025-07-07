package benchmark

import (
	"testing"

	"github.com/osmike/fcache"
)

func BenchmarkCachedParallel(b *testing.B) {
	const delay = 10
	cached := fcache.NewCachedFunction(slowFunc, nil, nil)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// All goroutines use the same key to test in-flight deduplication under high concurrency
			_, err := cached(delay)
			if err != nil {
				b.Fatalf("err: %v", err)
			}
		}
	})
}
