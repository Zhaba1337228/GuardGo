package guardgo

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestReason_String(t *testing.T) {
	t.Parallel()
	cases := map[Reason]string{
		ReasonAllowed:          "allowed",
		ReasonRateLimited:      "rate_limited",
		ReasonBlacklisted:      "blacklisted",
		ReasonRedisError:       "redis_error",
		ReasonFailOpen:         "fail_open",
		ReasonPenaltyForbidden: "penalty_forbidden",
		Reason(99):             "unknown",
	}
	for r, want := range cases {
		if got := r.String(); got != want {
			t.Errorf("Reason(%d).String() = %q, want %q", r, got, want)
		}
	}
}

func TestDecision_StatusCodeOr(t *testing.T) {
	t.Parallel()

	if got := (Decision{StatusCode: 418}).StatusCodeOr(200); got != 418 {
		t.Errorf("explicit status: want 418, got %d", got)
	}
	if got := (Decision{}).StatusCodeOr(403); got != 403 {
		t.Errorf("fallback default: want 403, got %d", got)
	}
	if got := (Decision{}).StatusCodeOr(0); got != http.StatusTooManyRequests {
		t.Errorf("zero default → 429, got %d", got)
	}
}

func TestDecision_Headers(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	d := Decision{Limit: 200, Remaining: 17, ResetAt: now}
	h := d.Headers()

	if h["X-RateLimit-Limit"] != "200" {
		t.Errorf("Limit header: %q", h["X-RateLimit-Limit"])
	}
	if h["X-RateLimit-Remaining"] != "17" {
		t.Errorf("Remaining header: %q", h["X-RateLimit-Remaining"])
	}
	if h["X-RateLimit-Reset"] != strconv.FormatInt(now.Unix(), 10) {
		t.Errorf("Reset header: %q", h["X-RateLimit-Reset"])
	}
	if h["X-RateLimit-Reset-At"] != now.Format(time.RFC3339) {
		t.Errorf("Reset-At header: %q", h["X-RateLimit-Reset-At"])
	}
}

func TestNewDecisionWithTTL_FallsBackToWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	d := newDecisionWithTTL(true, ReasonAllowed, 0, 100, now, 0, 5*time.Second)
	want := now.Add(5 * time.Second)
	if !d.ResetAt.Equal(want) {
		t.Errorf("ResetAt: want %s, got %s", want, d.ResetAt)
	}
	if d.Remaining != 100 {
		t.Errorf("Remaining: want 100, got %d", d.Remaining)
	}
}

func TestNewDecisionWithTTL_NegativeRemainingClamped(t *testing.T) {
	t.Parallel()
	now := time.Now()
	d := newDecisionWithTTL(false, ReasonRateLimited, 200, 100, now, 1000, time.Second)
	if d.Remaining != 0 {
		t.Errorf("Remaining must clamp to 0, got %d", d.Remaining)
	}
}
