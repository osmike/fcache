package benchmark

import "testing"

func BenchmarkDirect(b *testing.B) {
	const delay = 10
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := slowFunc(delay)
		if err != nil {
			b.Fatalf("err: %v", err)
		}
	}
}
