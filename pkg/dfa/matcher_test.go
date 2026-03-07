package dfa

import "testing"

func TestMatcherFindAll(t *testing.T) {
	matcher := New([]Rule{
		{ID: "sqli", Weight: 10, Token: "select"},
		{ID: "drop", Weight: 15, Token: "drop"},
		{ID: "union", Weight: 8, Token: "union"},
	})

	matches := matcher.FindAll("q=UNION+SELECT+1")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}
