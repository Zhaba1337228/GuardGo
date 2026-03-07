package bloom

import "testing"

func TestFilterAddAndTest(t *testing.T) {
	filter := New(1024, 4)
	ip := "192.168.1.42"

	if filter.Test(ip) {
		t.Fatalf("expected empty bloom filter to return false")
	}

	filter.Add(ip)
	if !filter.Test(ip) {
		t.Fatalf("expected bloom filter to return true for added item")
	}
}
