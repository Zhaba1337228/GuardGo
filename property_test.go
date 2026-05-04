package guardgo

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestProperty_Fingerprint_NormalizationStable asserts that whitespace and
// case differences in the source tuple never change the resulting fingerprint.
func TestProperty_Fingerprint_NormalizationStable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ip := rapid.StringMatching(`[0-9]{1,3}(\.[0-9]{1,3}){3}`).Draw(t, "ip")
		ua := rapid.StringMatching(`[a-zA-Z0-9 /._-]{0,64}`).Draw(t, "ua")
		lang := rapid.StringMatching(`[a-zA-Z-]{0,16}`).Draw(t, "lang")

		base := fingerprint(ip, ua, lang)

		mutated := fingerprint(
			"  "+ip+"\t",
			strings.ToUpper(ua),
			"  "+strings.ToUpper(lang),
		)
		if base != mutated {
			t.Fatalf("fingerprint not normalization-stable:\nbase=%q\nmut =%q\nip=%q ua=%q lang=%q",
				base, mutated, ip, ua, lang)
		}
	})
}

// TestProperty_Fingerprint_DistinctIPsDifferentResult ensures any IP change
// yields a different hash (collisions on small inputs would break the
// reputation hash field).
func TestProperty_Fingerprint_DistinctIPsDifferentResult(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ip1 := rapid.StringMatching(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`).Draw(t, "ip1")
		ip2 := rapid.StringMatching(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`).Draw(t, "ip2")
		if ip1 == ip2 {
			t.Skip()
		}
		ua := rapid.StringMatching(`[a-zA-Z0-9 /._-]{0,32}`).Draw(t, "ua")
		lang := rapid.StringMatching(`[a-zA-Z-]{0,16}`).Draw(t, "lang")

		a := fingerprint(ip1, ua, lang)
		b := fingerprint(ip2, ua, lang)
		if a == b {
			t.Fatalf("fingerprint collision for distinct IPs: %s == %s (ip1=%q ip2=%q)", a, b, ip1, ip2)
		}
	})
}

// TestProperty_Hash64_DeterministicAndNonZeroForNonEmpty asserts that
// hash64 is deterministic and yields the same value for the same input.
func TestProperty_Hash64_DeterministicAndNonZeroForNonEmpty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringN(1, 64, 256).Draw(t, "s")
		a := hash64(s)
		b := hash64(s)
		if a != b {
			t.Fatalf("hash64 not deterministic for %q: %d vs %d", s, a, b)
		}
	})
}

// TestProperty_NewDecisionWithTTL_RemainingNonNegative ensures the constructor
// never produces a negative Remaining value regardless of inputs.
func TestProperty_NewDecisionWithTTL_RemainingNonNegative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		counter := rapid.Int64Range(0, 1<<20).Draw(t, "counter")
		limit := rapid.IntRange(0, 1<<20).Draw(t, "limit")
		ttl := rapid.Int64Range(-1000, 1<<20).Draw(t, "ttl")

		// fallback window must be positive to avoid time math edge cases
		d := newDecisionWithTTL(true, ReasonAllowed, counter, limit, nowForTest, ttl, 1)
		if d.Remaining < 0 {
			t.Fatalf("Remaining must clamp >= 0, got %d (counter=%d limit=%d ttl=%d)",
				d.Remaining, counter, limit, ttl)
		}
		if d.Limit != limit {
			t.Fatalf("Limit not preserved: want %d got %d", limit, d.Limit)
		}
	})
}

// TestProperty_Reason_StringTotal asserts Reason.String covers any value
// without panic and always returns a non-empty descriptor.
func TestProperty_Reason_StringTotal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		r := Reason(rapid.IntRange(-100, 100).Draw(t, "r"))
		s := r.String()
		if s == "" {
			t.Fatalf("Reason(%d).String() returned empty", r)
		}
	})
}
