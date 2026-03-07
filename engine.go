package guardgo

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
	"guardgo/internal/blacklist"
	"guardgo/internal/redislua"
	"guardgo/pkg/bloom"
)

type Engine struct {
	cfg MiddlewareConfig

	cache *blacklist.Cache
	bf    *bloom.Filter
	sh    *selfHealingState

	bgStop chan struct{}
	bgWg   sync.WaitGroup

	reqPool sync.Pool

	analysisCh chan *requestSnapshot
}

// Config returns effective engine configuration with defaults applied.
func (e *Engine) Config() MiddlewareConfig {
	return e.cfg
}

type requestSnapshot struct {
	ip     string
	key    string
	method string
	host   string
	path   string
	rawQ   string
	hdr    http.Header
	ts     time.Time
}

func NewEngine(cfg MiddlewareConfig) (*Engine, error) {
	cfg = cfg.withDefaults()
	if cfg.Redis == nil {
		cfg.Redis = DefaultConfig().Redis
	}
	c, err := blacklist.New(cfg.CacheSize)
	if err != nil {
		return nil, err
	}

	e := &Engine{
		cfg:    cfg,
		cache:  c,
		bgStop: make(chan struct{}),
		reqPool: sync.Pool{
			New: func() any { return new(requestSnapshot) },
		},
	}
	if cfg.Bloom.Enabled {
		e.bf = bloom.New(cfg.Bloom.Bits, cfg.Bloom.Hashes)
	}
	if cfg.SelfHealing.Enabled {
		e.sh = newSelfHealingState(cfg.SelfHealing)
	}

	if cfg.AdaptiveBan.Enabled && len(cfg.Rules) > 0 {
		workers := cfg.AdaptiveBan.Workers
		if workers <= 0 {
			workers = 2 * runtime.GOMAXPROCS(0)
		}
		e.analysisCh = make(chan *requestSnapshot, cfg.AdaptiveBan.QueueSize)
		e.startBackground(workers)
	}

	return e, nil
}

func (e *Engine) Close() {
	if e.analysisCh == nil {
		return
	}
	close(e.bgStop)
	e.bgWg.Wait()
}

func (e *Engine) startBackground(workers int) {
	for range workers {
		e.bgWg.Add(1)
		go func() {
			defer e.bgWg.Done()
			e.bgWorker()
		}()
	}
}

func (e *Engine) bgWorker() {
	for {
		select {
		case <-e.bgStop:
			return
		case snap := <-e.analysisCh:
			if snap == nil {
				continue
			}
			e.processSnapshot(snap)
			e.putSnapshot(snap)
		}
	}
}

func (e *Engine) processSnapshot(s *requestSnapshot) {
	r := &http.Request{
		Method: s.method,
		Host:   s.host,
		URL: &url.URL{
			Path:     s.path,
			RawQuery: s.rawQ,
		},
		Header: s.hdr,
	}
	weight := 0
	for _, rule := range e.cfg.Rules {
		weight += rule.Evaluate(r)
	}
	if weight <= 0 {
		return
	}

	// Adaptive ban: increment per-IP threat counter, with TTL window.
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	prefix := e.cfg.Prefix
	ip := s.ip
	threatKey := prefix + ":threat:" + ip

	pipe := e.cfg.Redis.Pipeline()
	incr := pipe.Incr(ctx, threatKey)
	pipe.PExpire(ctx, threatKey, e.cfg.AdaptiveBan.ThreatKeyWindow)
	_, err := pipe.Exec(ctx)
	if err != nil {
		e.cfg.Metrics.IncRedisErrors()
		e.recordStat("redis_errors_total", 1, map[string]string{"op": "adaptive_ban_increment"})
		if e.cfg.OnError != nil {
			e.cfg.OnError(err)
		}
		return
	}
	if incr.Val() < e.cfg.AdaptiveBan.ThreatCountThreshold {
		return
	}

	_ = e.banIP(ctx, ip, e.cfg.AdaptiveBan.BanTTL)
}

func (e *Engine) getSnapshot() *requestSnapshot {
	s := e.reqPool.Get().(*requestSnapshot)
	*s = requestSnapshot{}
	return s
}

func (e *Engine) putSnapshot(s *requestSnapshot) {
	if s.hdr != nil {
		for k := range s.hdr {
			delete(s.hdr, k)
		}
	}
	e.reqPool.Put(s)
}

func (e *Engine) snapshotFromRequest(r *http.Request, ip, key string, now time.Time) *requestSnapshot {
	s := e.getSnapshot()
	s.ip = ip
	s.key = key
	s.method = r.Method
	s.host = r.Host
	if r.URL != nil {
		s.path = r.URL.Path
		s.rawQ = r.URL.RawQuery
	}

	if len(e.cfg.AdaptiveBan.CaptureHeaders) > 0 {
		if s.hdr == nil {
			s.hdr = make(http.Header, len(e.cfg.AdaptiveBan.CaptureHeaders))
		}
		for _, name := range e.cfg.AdaptiveBan.CaptureHeaders {
			if v := r.Header.Values(name); len(v) > 0 {
				// Copy to avoid aliasing caller's slice.
				dst := make([]string, len(v))
				copy(dst, v)
				s.hdr[name] = dst
			}
		}
	}

	s.ts = now
	return s
}

func defaultIPFunc(r *http.Request) string {
	ra := r.RemoteAddr
	if ra == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(ra)
	if err == nil {
		return host
	}
	// Might already be host-only.
	return ra
}

func (e *Engine) redisKeysFor(ip, key string) (blKey, counterKey string) {
	p := e.cfg.Prefix
	blKey = p + ":bl:" + ip
	counterKey = p + ":rl:" + key
	return blKey, counterKey
}

func (e *Engine) shouldEnqueueAnalysis() bool {
	return e.analysisCh != nil
}

// Check performs critical-path checks:
// - local blacklist cache
// - Redis Lua (blacklist + rate-limit counter)
//
// It returns allowed, reason code and current counter value.
func (e *Engine) Check(ctx context.Context, r *http.Request) (allowed bool, reason Reason, counter int64) {
	d := e.Process(ctx, r)
	return d.Allowed, d.Reason, d.Counter
}

// Process runs request through guard engine and returns a full decision payload.
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
		d := newDecision(false, ReasonBlacklisted, 0, effectiveLimit, now, e.cfg.Window)
		e.finishDecision(now, span, d, redisDurationMs)
		return d
	}
	if ip != "" && e.bf != nil && !e.bf.Test(ip) {
		e.cfg.Metrics.IncAllowed()
		e.recordRequestStat("allowed", "bloom_negative")
		if e.shouldEnqueueAnalysis() {
			e.tryEnqueue(r, ip, key, now)
		}
		d := newDecision(true, ReasonAllowed, 0, effectiveLimit, now, e.cfg.Window)
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
			d := newDecision(true, ReasonFailOpen, 0, effectiveLimit, now, e.cfg.Window)
			e.finishDecision(now, span, d, redisDurationMs)
			return d
		}
		e.recordRequestStat("blocked", "redis_error")
		d := newDecision(false, ReasonRedisError, 0, effectiveLimit, now, e.cfg.Window)
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
	// go-redis returns []interface{} with int64 values.
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

func (e *Engine) banIP(ctx context.Context, ip string, ttl time.Duration) error {
	if ip == "" {
		return nil
	}
	blKey, _ := e.redisKeysFor(ip, ip)

	pipe := e.cfg.Redis.Pipeline()
	pipe.SAdd(ctx, blKey, "1")
	pipe.PExpire(ctx, blKey, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		e.cfg.Metrics.IncRedisErrors()
		e.recordStat("redis_errors_total", 1, map[string]string{"op": "ban_ip"})
		if e.cfg.OnError != nil {
			e.cfg.OnError(err)
		}
		return err
	}
	e.cache.Blacklist(ip, time.Now().Add(ttl))
	if e.bf != nil {
		e.bf.Add(ip)
	}
	return nil
}

func (e *Engine) recordRequestStat(status, source string) {
	e.recordStat("requests_total", 1, map[string]string{
		"status": status,
		"source": source,
	})
}

func (e *Engine) recordStat(name string, value float64, labels map[string]string) {
	e.cfg.Stats.Record(name, value, labels)
}

// Reason describes the decision made by Engine.Check.
type Reason int

const (
	ReasonAllowed Reason = iota
	ReasonRateLimited
	ReasonBlacklisted
	ReasonRedisError
	ReasonFailOpen
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
	default:
		return "unknown"
	}
}

// Decision is a normalized engine output used by all middleware adapters.
type Decision struct {
	Allowed   bool
	Reason    Reason
	Counter   int64
	Limit     int
	Remaining int
	ResetAt   time.Time
}

func newDecision(allowed bool, reason Reason, counter int64, limit int, now time.Time, fallbackWindow time.Duration) Decision {
	return newDecisionWithTTL(allowed, reason, counter, limit, now, int64(fallbackWindow/time.Millisecond), fallbackWindow)
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
	remaining := limit - int(counter)
	if remaining < 0 {
		remaining = 0
	}
	return Decision{
		Allowed:   allowed,
		Reason:    reason,
		Counter:   counter,
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   now.Add(time.Duration(ttlMillis) * time.Millisecond),
	}
}
