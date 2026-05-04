package guardgo

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/Zhaba1337228/GuardGo/internal/redislua"
)

// Check performs critical-path checks:
//   - local blacklist cache
//   - Redis Lua (blacklist + rate-limit counter)
//
// It returns allowed, reason code and current counter value.
func (e *Engine) Check(ctx context.Context, r *http.Request) (allowed bool, reason Reason, counter int64) {
	d := e.Process(ctx, r)
	return d.Allowed, d.Reason, d.Counter
}

// Process runs the request through the guard engine and returns a full
// decision payload that all middleware adapters can consume uniformly.
func (e *Engine) Process(ctx context.Context, r *http.Request) Decision {
	now := time.Now()
	ip := e.cfg.IPFunc(r)
	key := e.cfg.KeyFunc(r)
	if key == "" {
		key = ip
	}
	ctx, span := e.startProcessSpan(ctx, r, ip, key)
	defer span.End()

	effectiveLimit := e.cfg.MaxRequests
	if e.sh != nil {
		effectiveLimit = e.sh.effectiveLimit(ip, e.cfg.MaxRequests, now)
	}
	redisDurationMs := 0.0

	if ip != "" && e.cache.IsBlacklisted(ip, now) {
		e.cfg.Metrics.IncBlacklisted()
		e.recordRequestStat("blocked", "cache_blacklist")
		d := newDecision(false, ReasonBlacklisted, effectiveLimit, now, e.cfg.Window)
		e.finishDecision(now, span, d, redisDurationMs)
		return d
	}
	if ip != "" {
		inPenalty, penaltyTTL := e.isInPenalty(ctx, ip, now)
		if inPenalty {
			if !e.passesPenaltyChecks(r) {
				banTTL := e.cfg.Behavioral.PenaltyBanTTL
				if e.cfg.Reputation.Enabled {
					banTTL = e.cfg.Reputation.BanLevel3
				}
				_ = e.banIP(ctx, ip, banTTL)
				e.recordRequestStat("blocked", "penalty_forbidden")
				d := newDecisionWithTTL(false, ReasonPenaltyForbidden, 0, 1, now, int64(banTTL/time.Millisecond), e.cfg.Window)
				d.StatusCode = http.StatusForbidden
				e.finishDecision(now, span, d, redisDurationMs)
				return d
			}
			if effectiveLimit > 1 {
				effectiveLimit = 1
			}
			if penaltyTTL > 0 {
				e.recordStat("penalty_box_hits_total", 1, map[string]string{"mode": "strict"})
			}
		}
	}
	if ip != "" && e.bf != nil && !e.bf.Test(ip) {
		e.cfg.Metrics.IncAllowed()
		e.recordRequestStat("allowed", "bloom_negative")
		if e.shouldEnqueueAnalysis() {
			e.tryEnqueue(r, ip, key, now)
		}
		d := newDecision(true, ReasonAllowed, effectiveLimit, now, e.cfg.Window)
		e.finishDecision(now, span, d, redisDurationMs)
		return d
	}

	blKey, counterKey := e.redisKeysFor(ip, key)

	start := time.Now()
	res, err := redislua.CriticalPathScript.Run(ctx, e.cfg.Redis, []string{blKey, counterKey}, e.cfg.Window.Milliseconds(), effectiveLimit).Result()
	redisDuration := time.Since(start)
	redisDurationMs = float64(redisDuration.Microseconds()) / 1000.0
	e.cfg.Metrics.ObserveLuaLatency(redisDuration)
	if err != nil {
		e.cfg.Metrics.IncRedisErrors()
		e.recordStat("redis_errors_total", 1, map[string]string{"op": "critical_path_lua"})
		if e.cfg.OnError != nil {
			e.cfg.OnError(err)
		}
		if e.cfg.FailOpen {
			e.cfg.Metrics.IncAllowed()
			e.recordRequestStat("allowed", "fail_open")
			if e.shouldEnqueueAnalysis() {
				e.tryEnqueue(r, ip, key, now)
			}
			d := newDecision(true, ReasonFailOpen, effectiveLimit, now, e.cfg.Window)
			e.finishDecision(now, span, d, redisDurationMs)
			return d
		}
		e.recordRequestStat("blocked", "redis_error")
		d := newDecision(false, ReasonRedisError, effectiveLimit, now, e.cfg.Window)
		e.finishDecision(now, span, d, redisDurationMs)
		return d
	}

	code, counter, ttl := parseLuaResult(res)
	switch code {
	case 2:
		e.cfg.Metrics.IncBlacklisted()
		e.recordRequestStat("blocked", "redis_blacklist")
		// Cache blacklisted for a short period to avoid hammering Redis.
		e.cache.Blacklist(ip, now.Add(2*time.Second))
		if ip != "" && e.bf != nil {
			e.bf.Add(ip)
		}
		d := newDecisionWithTTL(false, ReasonBlacklisted, counter, effectiveLimit, now, ttl, e.cfg.Window)
		e.finishDecision(now, span, d, redisDurationMs)
		return d
	case 1:
		e.cfg.Metrics.IncRateLimited()
		e.recordRequestStat("blocked", "rate_limit")
		d := newDecisionWithTTL(false, ReasonRateLimited, counter, effectiveLimit, now, ttl, e.cfg.Window)
		e.finishDecision(now, span, d, redisDurationMs)
		return d
	default:
		e.cfg.Metrics.IncAllowed()
		e.recordRequestStat("allowed", "lua_allow")
		if e.shouldEnqueueAnalysis() {
			e.tryEnqueue(r, ip, key, now)
		}
		d := newDecisionWithTTL(true, ReasonAllowed, counter, effectiveLimit, now, ttl, e.cfg.Window)
		e.finishDecision(now, span, d, redisDurationMs)
		return d
	}
}

func (e *Engine) finishDecision(now time.Time, span trace.Span, d Decision, redisDurationMs float64) {
	if e.sh != nil {
		e.sh.observe(d, now)
	}
	e.annotateDecision(span, d, redisDurationMs)
}

func (e *Engine) tryEnqueue(r *http.Request, ip, key string, now time.Time) {
	snap := e.snapshotFromRequest(r, ip, key, now)
	select {
	case e.analysisCh <- snap:
	default:
		// Drop under pressure; async analysis is best-effort.
		e.putSnapshot(snap)
	}
}

func parseLuaResult(v any) (code int64, counter int64, ttlMillis int64) {
	arr, ok := v.([]any)
	if !ok || len(arr) < 2 {
		return 0, 0, 0
	}
	if c, ok := arr[0].(int64); ok {
		code = c
	}
	if cur, ok := arr[1].(int64); ok {
		counter = cur
	}
	if len(arr) > 2 {
		if ttl, ok := arr[2].(int64); ok {
			ttlMillis = ttl
		}
	}
	return code, counter, ttlMillis
}
