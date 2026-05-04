package dfa

import (
	"strings"
	"testing"
)

// FuzzMatcher_New ensures that arbitrary rule input never panics New(),
// produces a usable Matcher, and always returns Match.RuleID values that
// resolve back to the original rule set.
func FuzzMatcher_New(f *testing.F) {
	seeds := [][3]string{
		{"r1", "1", "select"},
		{"r2", "5", "UNION SELECT"},
		{"", "0", ""},
		{"r3", "-7", "  WhItEsPaCe  "},
		{"r4", "1", "\x00\x01binary"},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1], s[2])
	}

	f.Fuzz(func(t *testing.T, id, weight, token string) {
		w := int64(0)
		for _, c := range weight {
			w = w*10 + int64(c-'0')
			if w > 1<<32 || w < -(1<<32) {
				w = 1
				break
			}
		}
		rules := []Rule{{ID: id, Weight: w, Token: token}}

		m := New(rules)
		if m == nil {
			t.Fatal("New returned nil")
		}

		// Searching anywhere with the same matcher must not panic
		// even with arbitrary haystacks.
		_ = m.FindAll("")
		_ = m.FindAll(strings.Repeat(token, 3))
		_ = m.FindAll(token + "tail")

		// All matches must reference an existing rule by ID.
		for _, hit := range m.FindAll(token) {
			found := false
			for _, r := range rules {
				if hit.RuleID == r.ID || hit.RuleID == strings.TrimSpace(strings.ToLower(r.ID)) {
					found = true
					break
				}
			}
			if !found && len(strings.TrimSpace(strings.ToLower(token))) > 0 && id != "" {
				t.Fatalf("match returned unknown RuleID=%q for token=%q", hit.RuleID, token)
			}
		}
	})
}

// FuzzMatcher_FindAll exercises the search loop against arbitrary haystacks.
func FuzzMatcher_FindAll(f *testing.F) {
	f.Add("select", "users WHERE id=1 union SELECT 1")
	f.Add("/etc/passwd", "/?file=../../etc/passwd")
	f.Add("", "anything")
	f.Add("\x00", "\x00\x00\x00")

	f.Fuzz(func(t *testing.T, token, haystack string) {
		m := New([]Rule{{ID: "r", Weight: 1, Token: token}})
		matches := m.FindAll(haystack)

		// Each returned match must point to the only configured rule.
		for _, hit := range matches {
			if hit.RuleID != "r" {
				t.Fatalf("unexpected RuleID=%q", hit.RuleID)
			}
			if hit.Weight <= 0 {
				t.Fatalf("non-positive weight=%d", hit.Weight)
			}
		}
	})
}
