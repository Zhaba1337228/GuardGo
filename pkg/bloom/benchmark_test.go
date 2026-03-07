package bloom

import "testing"

func BenchmarkBloomCleanRequest(b *testing.B) {
	filter := New(1<<20, 6)
	needle := "203.0.113.77"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filter.Test(needle)
	}
}
