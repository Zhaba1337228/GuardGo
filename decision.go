package guardgo

import (
	"net/http"
	"strconv"
	"time"
)

// Reason describes the decision made by Engine.Check / Engine.Process.
type Reason int

const (
	ReasonAllowed Reason = iota
	ReasonRateLimited
	ReasonBlacklisted
	ReasonRedisError
	ReasonFailOpen
	ReasonPenaltyForbidden
)

func (r Reason) String() string {
	switch r {
	case ReasonAllowed:
		return "allowed"
	case ReasonRateLimited:
		return "rate_limited"
	case ReasonBlacklisted:
		return "blacklisted"
	case ReasonRedisError:
		return "redis_error"
	case ReasonFailOpen:
		return "fail_open"
	case ReasonPenaltyForbidden:
		return "penalty_forbidden"
	default:
		return "unknown"
	}
}

// Decision is a normalized engine output used by all middleware adapters.
type Decision struct {
	Allowed    bool
	Reason     Reason
	Counter    int64
	Limit      int
	Remaining  int
	ResetAt    time.Time
	StatusCode int
}

func newDecision(allowed bool, reason Reason, limit int, now time.Time, fallbackWindow time.Duration) Decision {
	return newDecisionWithTTL(allowed, reason, 0, limit, now, int64(fallbackWindow/time.Millisecond), fallbackWindow)
}

func newDecisionWithTTL(
	allowed bool,
	reason Reason,
	counter int64,
	limit int,
	now time.Time,
	ttlMillis int64,
	fallbackWindow time.Duration,
) Decision {
	if ttlMillis <= 0 {
		ttlMillis = int64(fallbackWindow / time.Millisecond)
	}
	remaining := max(limit-int(counter), 0)
	return Decision{
		Allowed:   allowed,
		Reason:    reason,
		Counter:   counter,
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   now.Add(time.Duration(ttlMillis) * time.Millisecond),
	}
}

func (d Decision) StatusCodeOr(defaultCode int) int {
	if d.StatusCode != 0 {
		return d.StatusCode
	}
	if defaultCode != 0 {
		return defaultCode
	}
	return http.StatusTooManyRequests
}

func (d Decision) Headers() map[string]string {
	return map[string]string{
		"X-RateLimit-Limit":     strconv.Itoa(d.Limit),
		"X-RateLimit-Remaining": strconv.Itoa(d.Remaining),
		"X-RateLimit-Reset":     strconv.FormatInt(d.ResetAt.Unix(), 10),
		"X-RateLimit-Reset-At":  d.ResetAt.UTC().Format(time.RFC3339),
	}
}
