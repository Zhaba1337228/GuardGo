package integration_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"guardgo"
	"guardgo/rules"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type countingMetrics struct {
	allowed     int64
	rateLimited int64
	blacklisted int64
	redisErrors int64
	latencyObs  int64
}

func (m *countingMetrics) IncAllowed() {
	atomic.AddInt64(&m.allowed, 1)
}

func (m *countingMetrics) IncRateLimited() {
	atomic.AddInt64(&m.rateLimited, 1)
}

func (m *countingMetrics) IncBlacklisted() {
	atomic.AddInt64(&m.blacklisted, 1)
}

func (m *countingMetrics) IncRedisErrors() {
	atomic.AddInt64(&m.redisErrors, 1)
}

func (m *countingMetrics) ObserveLuaLatency(_ time.Duration) {
	atomic.AddInt64(&m.latencyObs, 1)
}

func (m *countingMetrics) snapshot() (allowed, rateLimited, blacklisted, redisErrors, latencyObs int64) {
	return atomic.LoadInt64(&m.allowed),
		atomic.LoadInt64(&m.rateLimited),
		atomic.LoadInt64(&m.blacklisted),
		atomic.LoadInt64(&m.redisErrors),
		atomic.LoadInt64(&m.latencyObs)
}

func TestEngineBehavior(t *testing.T) {
	t.Run("rate_limit_resets_after_window", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 2,
			Window:      100 * time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/rate-reset", nil)
		req.RemoteAddr = "10.1.1.1:1234"
		ctx := context.Background()

		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)
		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)
		assertCheck(t, engine, ctx, req, false, guardgo.ReasonRateLimited)

		mr.FastForward(150 * time.Millisecond)
		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)
		t.Log("window reset validated: request allowed after TTL expiration")
	})

	t.Run("redis_blacklist_blocks_request", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 100,
			Window:      time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/blocked", nil)
		req.RemoteAddr = "10.2.2.2:5555"
		ctx := context.Background()

		if _, err := mr.SAdd("guardgo:bl:10.2.2.2", "1"); err != nil {
			t.Fatal(err)
		}
		mr.SetTTL("guardgo:bl:10.2.2.2", time.Minute)

		assertCheck(t, engine, ctx, req, false, guardgo.ReasonBlacklisted)
		assertCheck(t, engine, ctx, req, false, guardgo.ReasonBlacklisted)
		t.Log("redis blacklist and local cache path validated")
	})

	t.Run("fail_open_allows_when_redis_unavailable", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{
			Addr:       mr.Addr(),
			MaxRetries: 0,
		})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
			FailOpen:    true,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/fail-open", nil)
		req.RemoteAddr = "10.3.3.3:6789"
		ctx := context.Background()

		mr.Close()

		ok, reason, _ := engine.Check(ctx, req)
		if !ok || reason != guardgo.ReasonFailOpen {
			t.Fatalf("expected fail-open allow, got ok=%v reason=%s", ok, reason.String())
		}
		t.Log("fail-open path validated")
	})

	t.Run("fail_close_denies_when_redis_unavailable", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{
			Addr:       mr.Addr(),
			MaxRetries: 0,
		})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
			FailOpen:    false,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/fail-close", nil)
		req.RemoteAddr = "10.4.4.4:6789"
		ctx := context.Background()
		mr.Close()

		ok, reason, _ := engine.Check(ctx, req)
		if ok || reason != guardgo.ReasonRedisError {
			t.Fatalf("expected fail-close deny, got ok=%v reason=%s", ok, reason.String())
		}
		t.Log("fail-close path validated")
	})

	t.Run("custom_key_func_is_used", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
			KeyFunc: func(r *http.Request) string {
				return r.URL.Path
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		ctx := context.Background()
		reqA, _ := http.NewRequest("GET", "http://example.com/a", nil)
		reqA.RemoteAddr = "10.5.5.5:1111"
		reqB, _ := http.NewRequest("GET", "http://example.com/b", nil)
		reqB.RemoteAddr = "10.5.5.5:1111"

		assertCheck(t, engine, ctx, reqA, true, guardgo.ReasonAllowed)
		assertCheck(t, engine, ctx, reqA, false, guardgo.ReasonRateLimited)
		assertCheck(t, engine, ctx, reqB, true, guardgo.ReasonAllowed)
		t.Log("custom key function validated")
	})

	t.Run("adaptive_ban_blacklists_after_threat_threshold", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 100,
			Window:      time.Second,
			Rules: []guardgo.Rule{
				rules.QueryContains{Tokens: []string{"sqli"}, Weight: 5},
			},
			AdaptiveBan: guardgo.AdaptiveBanConfig{
				Enabled:              true,
				ThreatCountThreshold: 1,
				ThreatKeyWindow:      time.Second,
				BanTTL:               time.Minute,
				Workers:              1,
				QueueSize:            32,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/login?q=sqli", nil)
		req.RemoteAddr = "10.6.6.6:3131"
		ctx := context.Background()

		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)

		deadline := time.Now().Add(500 * time.Millisecond)
		for {
			ok, reason, _ := engine.Check(ctx, req)
			if !ok && reason == guardgo.ReasonBlacklisted {
				t.Log("adaptive ban promoted ip to blacklist")
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("expected adaptive ban to blacklist request before timeout")
			}
			time.Sleep(10 * time.Millisecond)
		}
	})

	t.Run("metrics_counts_are_updated", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		metrics := &countingMetrics{}
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
			Metrics:     metrics,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/metrics", nil)
		req.RemoteAddr = "10.7.7.7:9999"
		ctx := context.Background()

		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)
		assertCheck(t, engine, ctx, req, false, guardgo.ReasonRateLimited)

		allowed, rateLimited, blacklisted, redisErrors, latencyObs := metrics.snapshot()
		if allowed != 1 || rateLimited != 1 || blacklisted != 0 || redisErrors != 0 || latencyObs < 2 {
			t.Fatalf(
				"unexpected metrics state: allowed=%d rate_limited=%d blacklisted=%d redis_errors=%d latency_obs=%d",
				allowed, rateLimited, blacklisted, redisErrors, latencyObs,
			)
		}
		t.Logf(
			"metrics validated: allowed=%d rate_limited=%d blacklisted=%d redis_errors=%d latency_obs=%d",
			allowed, rateLimited, blacklisted, redisErrors, latencyObs,
		)
	})

	t.Run("bloom_negative_short_circuits_redis", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
			Bloom: guardgo.BloomConfig{
				Enabled: true,
				Bits:    1024,
				Hashes:  4,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/bloom", nil)
		req.RemoteAddr = "10.8.8.8:1000"
		ctx := context.Background()

		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)
		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)
		t.Log("bloom negative path validated: redis rate-limit path bypassed")
	})

	t.Run("stats_collector_receives_events", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		stats := &countingStats{}
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
			Stats:       stats,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		req, _ := http.NewRequest("GET", "http://example.com/stats", nil)
		req.RemoteAddr = "10.9.9.9:2000"
		ctx := context.Background()

		assertCheck(t, engine, ctx, req, true, guardgo.ReasonAllowed)
		assertCheck(t, engine, ctx, req, false, guardgo.ReasonRateLimited)

		if stats.count("requests_total", "status", "allowed") < 1 {
			t.Fatalf("expected at least one allowed request event in stats collector")
		}
		if stats.count("requests_total", "status", "blocked") < 1 {
			t.Fatalf("expected at least one blocked request event in stats collector")
		}
		t.Logf(
			"stats collector validated: allowed_events=%d blocked_events=%d",
			stats.count("requests_total", "status", "allowed"),
			stats.count("requests_total", "status", "blocked"),
		)
	})
}

func assertCheck(t *testing.T, engine *guardgo.Engine, ctx context.Context, req *http.Request, wantAllowed bool, wantReason guardgo.Reason) {
	t.Helper()
	gotAllowed, gotReason, gotCounter := engine.Check(ctx, req)
	if gotAllowed != wantAllowed || gotReason != wantReason {
		t.Fatalf(
			"unexpected check result for %s %s: allowed=%v reason=%s counter=%d, want allowed=%v reason=%s",
			req.Method, req.URL.String(), gotAllowed, gotReason.String(), gotCounter, wantAllowed, wantReason.String(),
		)
	}
}

type countingStats struct {
	mu     sync.Mutex
	events map[string]int64
}

func (c *countingStats) Record(name string, value float64, labels map[string]string) {
	_ = value
	status := labels["status"]
	source := labels["source"]
	key := name + "|" + status + "|" + source
	c.mu.Lock()
	if c.events == nil {
		c.events = make(map[string]int64, 4)
	}
	c.events[key]++
	c.mu.Unlock()
}

func (c *countingStats) count(metric, labelKey, labelValue string) int64 {
	if labelKey != "status" {
		return 0
	}
	prefix := metric + "|" + labelValue + "|"
	c.mu.Lock()
	defer c.mu.Unlock()
	var total int64
	for k, v := range c.events {
		if strings.HasPrefix(k, prefix) {
			total += v
		}
	}
	return total
}
