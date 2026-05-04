package guardgo

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func (e *Engine) redisKeysFor(ip, key string) (blKey, counterKey string) {
	p := e.cfg.Prefix
	blKey = p + ":bl:" + ip
	counterKey = p + ":rl:" + key
	return blKey, counterKey
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

func (e *Engine) penaltyKey(ip string) string {
	return e.cfg.Prefix + ":pbox:" + ip
}

func (e *Engine) isInPenalty(ctx context.Context, ip string, now time.Time) (bool, time.Duration) {
	if e.pbox.IsBlacklisted(ip, now) {
		if e.cfg.Reputation.Enabled {
			return true, e.cfg.Reputation.PenaltyTTL
		}
		return true, e.cfg.Behavioral.PenaltyTTL
	}
	ttl, err := e.cfg.Redis.PTTL(ctx, e.penaltyKey(ip)).Result()
	if err != nil || ttl <= 0 {
		return false, 0
	}
	e.pbox.Blacklist(ip, now.Add(ttl))
	return true, ttl
}

func (e *Engine) setPenalty(ctx context.Context, ip string, ttl time.Duration) error {
	if ip == "" {
		return nil
	}
	key := e.penaltyKey(ip)
	pipe := e.cfg.Redis.Pipeline()
	pipe.Set(ctx, key, "1", ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		e.recordStat("redis_errors_total", 1, map[string]string{"op": "set_penalty"})
		return err
	}
	e.pbox.Blacklist(ip, time.Now().Add(ttl))
	e.recordStat("penalty_box_entries_total", 1, map[string]string{"reason": "risk_threshold"})
	return nil
}

func (e *Engine) passesPenaltyChecks(r *http.Request) bool {
	if r == nil {
		return false
	}
	if e.cfg.Behavioral.RequireUserAgent && strings.TrimSpace(r.Header.Get("User-Agent")) == "" {
		return false
	}
	if e.cfg.Behavioral.RequireReferer && strings.TrimSpace(r.Header.Get("Referer")) == "" {
		return false
	}
	return true
}
