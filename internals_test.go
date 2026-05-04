package guardgo

import (
	"net/http"
	"strings"
	"testing"
)

func TestHash64_DeterministicAndDistinct(t *testing.T) {
	t.Parallel()
	a := hash64("foo")
	b := hash64("foo")
	c := hash64("bar")
	if a != b {
		t.Errorf("hash64 not deterministic: %d != %d", a, b)
	}
	if a == c {
		t.Errorf("hash64 collision for distinct inputs: foo=%d bar=%d", a, c)
	}
	if hash64("") == 0 {
		// fnv-style: empty string seeds offset, so result must be non-zero offset basis
		t.Logf("hash64(\"\") = 0 (not necessarily a bug, but unusual)")
	}
}

func TestFingerprint_LowercasesAndStable(t *testing.T) {
	t.Parallel()
	a := fingerprint("1.2.3.4", "Mozilla/5.0", "EN-us")
	b := fingerprint("  1.2.3.4 ", "mozilla/5.0", "en-us")
	if a != b {
		t.Errorf("fingerprint must normalize whitespace+case: %s vs %s", a, b)
	}
	if fingerprint("1.2.3.4", "x", "en") == fingerprint("1.2.3.5", "x", "en") {
		t.Error("fingerprint must change with IP")
	}
}

func TestRequestAnyText_NilSafe(t *testing.T) {
	t.Parallel()
	if got := requestAnyText(nil); got != "" {
		t.Errorf("nil request must yield empty string, got %q", got)
	}
}

func TestHeadersText_EmptyAndPopulated(t *testing.T) {
	t.Parallel()
	if got := headersText(nil); got != "" {
		t.Errorf("nil headers must yield empty, got %q", got)
	}
	h := http.Header{}
	h.Set("X-Test", "value1")
	h.Add("X-Test", "value2")
	got := headersText(h)
	if !strings.Contains(got, "X-Test:") || !strings.Contains(got, "value1") || !strings.Contains(got, "value2") {
		t.Errorf("headersText missing expected fragments: %q", got)
	}
}

func TestDefaultIPFunc(t *testing.T) {
	t.Parallel()
	cases := []struct {
		remote string
		want   string
	}{
		{"1.2.3.4:5050", "1.2.3.4"},
		{"[::1]:8080", "::1"},
		{"plain-host", "plain-host"},
		{"", ""},
	}
	for _, c := range cases {
		r := &http.Request{RemoteAddr: c.remote}
		if got := defaultIPFunc(r); got != c.want {
			t.Errorf("defaultIPFunc(%q) = %q, want %q", c.remote, got, c.want)
		}
	}
}

func TestParseLuaResult(t *testing.T) {
	t.Parallel()
	code, counter, ttl := parseLuaResult([]any{int64(2), int64(7), int64(1500)})
	if code != 2 || counter != 7 || ttl != 1500 {
		t.Errorf("got code=%d counter=%d ttl=%d", code, counter, ttl)
	}

	if c, n, tt := parseLuaResult(nil); c != 0 || n != 0 || tt != 0 {
		t.Errorf("nil input must yield zeros, got %d %d %d", c, n, tt)
	}
	if c, n, tt := parseLuaResult([]any{int64(1)}); c != 0 || n != 0 || tt != 0 {
		t.Errorf("short slice must yield zeros, got %d %d %d", c, n, tt)
	}
	if c, n, tt := parseLuaResult([]any{int64(0), int64(3)}); c != 0 || n != 3 || tt != 0 {
		t.Errorf("two-element slice: got %d %d %d", c, n, tt)
	}
}
