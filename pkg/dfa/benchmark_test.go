package dfa

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkDFA100RulesMatch(b *testing.B) {
	rules := make([]Rule, 0, 100)
	for i := 0; i < 100; i++ {
		rules = append(rules, Rule{
			ID:     fmt.Sprintf("r%d", i),
			Weight: 1,
			Token:  fmt.Sprintf("token_%03d", i),
		})
	}
	matcher := New(rules)
	payload := strings.Repeat("safe_payload_", 8) + "token_042" + strings.Repeat("_tail", 8)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.FindAll(payload)
	}
}
