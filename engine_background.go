package guardgo

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (e *Engine) startBackground(workers int) {
	for range workers {
		e.bgWg.Go(e.bgWorker)
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

// processSnapshot runs the rule pipeline asynchronously off the request
// critical path. It is best-effort: short Redis timeouts and dropped errors
// route through the configured OnError hook.
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
	score := 0.0
	for _, rule := range e.cfg.Rules {
		score += float64(rule.Evaluate(r))
	}
	for _, evaluator := range e.cfg.Evaluators {
		score += evaluator.CalculateScore(r)
	}
	if e.compiled != nil {
		score += float64(e.evaluateRuleset(r))
	}
	if e.behavior != nil {
		score += float64(e.behavior.observe(s.ip, r, s.ts))
	}

	if score <= 0 {
		return
	}
	e.recordStat("risk_score_add_total", score, map[string]string{"source": "pipeline"})

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	ip := s.ip
	if ip == "" {
		return
	}
	if e.cfg.Reputation.Enabled {
		action, banTTL, attempts, err := e.applyReputationScore(ctx, s, score)
		if err != nil {
			e.cfg.Metrics.IncRedisErrors()
			e.recordStat("redis_errors_total", 1, map[string]string{"op": "reputation_lua"})
			if e.cfg.OnError != nil {
				e.cfg.OnError(err)
			}
		} else {
			switch action {
			case 1:
				_ = e.setPenalty(ctx, ip, e.cfg.Reputation.PenaltyTTL)
			case 2:
				e.cache.Blacklist(ip, time.Now().Add(banTTL))
				if e.bf != nil {
					e.bf.Add(ip)
				}
				e.recordStat("reputation_blacklist_total", 1, map[string]string{
					"attempt": strconv.FormatInt(attempts, 10),
				})
			}
		}
	} else if score >= float64(e.cfg.Behavioral.ScoreThreshold) {
		_ = e.setPenalty(ctx, ip, e.cfg.Behavioral.PenaltyTTL)
	}

	if e.cfg.AdaptiveBan.Enabled {
		prefix := e.cfg.Prefix
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
		if incr.Val() >= e.cfg.AdaptiveBan.ThreatCountThreshold {
			_ = e.banIP(ctx, ip, e.cfg.AdaptiveBan.BanTTL)
		}
	}
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
	s.ua = r.Header.Get("User-Agent")
	s.lang = r.Header.Get("Accept-Language")

	if len(e.cfg.AdaptiveBan.CaptureHeaders) > 0 {
		if s.hdr == nil {
			s.hdr = make(http.Header, len(e.cfg.AdaptiveBan.CaptureHeaders))
		}
		for _, name := range e.cfg.AdaptiveBan.CaptureHeaders {
			if v := r.Header.Values(name); len(v) > 0 {
				dst := make([]string, len(v))
				copy(dst, v)
				s.hdr[name] = dst
			}
		}
	}

	s.ts = now
	return s
}
