package bloom

import (
	"sync/atomic"
)

// Filter is a lock-free bloom filter optimized for concurrent Add/Test.
type Filter struct {
	words []uint64
	bits  uint64
	k     uint8
}

func New(bits uint64, hashes uint8) *Filter {
	if bits == 0 {
		bits = 1 << 20
	}
	if hashes == 0 {
		hashes = 6
	}
	wordCount := (bits + 63) / 64
	return &Filter{
		words: make([]uint64, wordCount),
		bits:  bits,
		k:     hashes,
	}
}

func (f *Filter) Add(value string) {
	if value == "" {
		return
	}
	h1, h2 := twoHashes(value)
	for i := uint8(0); i < f.k; i++ {
		idx := (h1 + uint64(i)*h2) % f.bits
		wordIdx := idx / 64
		mask := uint64(1) << (idx % 64)
		for {
			old := atomic.LoadUint64(&f.words[wordIdx])
			if old&mask != 0 {
				break
			}
			if atomic.CompareAndSwapUint64(&f.words[wordIdx], old, old|mask) {
				break
			}
		}
	}
}

func (f *Filter) Test(value string) bool {
	if value == "" {
		return false
	}
	h1, h2 := twoHashes(value)
	for i := uint8(0); i < f.k; i++ {
		idx := (h1 + uint64(i)*h2) % f.bits
		wordIdx := idx / 64
		mask := uint64(1) << (idx % 64)
		if atomic.LoadUint64(&f.words[wordIdx])&mask == 0 {
			return false
		}
	}
	return true
}

func twoHashes(value string) (uint64, uint64) {
	const (
		offsetA = uint64(14695981039346656037)
		primeA  = uint64(1099511628211)
		offsetB = uint64(7809847782465536322)
		primeB  = uint64(1099511628211)
	)
	h1 := offsetA
	h2 := offsetB
	for i := 0; i < len(value); i++ {
		b := uint64(value[i])
		h1 ^= b
		h1 *= primeA
		h2 ^= (b + 0x9e3779b97f4a7c15)
		h2 *= primeB
	}
	if h2 == 0 {
		h2 = 0x9e3779b97f4a7c15
	}
	return h1, h2
}
