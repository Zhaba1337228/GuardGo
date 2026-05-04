package guardgo

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Zhaba1337228/GuardGo/internal/redislua"
)

func (e *Engine) applyReputationScore(ctx context.Context, s *requestSnapshot, score float64) (action int64, banTTL time.Duration, attempts int64, err error) {
	repKey := e.cfg.Prefix + ":rep"
	field := reputationField(s)
	penaltyKey := e.penaltyKey(s.ip)
	blKey, _ := e.redisKeysFor(s.ip, s.ip)
	attemptKey := e.cfg.Prefix + ":ban_attempts:" + s.ip

	res, runErr := redislua.ReputationScript.Run(
		ctx,
		e.cfg.Redis,
		[]string{repKey, penaltyKey, blKey, attemptKey},
		field,
		score,
		e.cfg.Reputation.ScoreWindow.Milliseconds(),
		e.cfg.Reputation.WarningLevel,
		e.cfg.Reputation.Threshold,
		e.cfg.Reputation.PenaltyTTL.Milliseconds(),
		e.cfg.Reputation.BackoffWindow.Milliseconds(),
		e.cfg.Reputation.BanLevel1.Milliseconds(),
		e.cfg.Reputation.BanLevel2.Milliseconds(),
		e.cfg.Reputation.BanLevel3.Milliseconds(),
	).Result()
	if runErr != nil {
		return 0, 0, 0, runErr
	}

	arr, ok := res.([]any)
	if !ok || len(arr) < 4 {
		return 0, 0, 0, nil
	}
	if v, ok := arr[1].(int64); ok {
		action = v
	}
	if v, ok := arr[2].(int64); ok {
		banTTL = time.Duration(v) * time.Millisecond
	}
	if v, ok := arr[3].(int64); ok {
		attempts = v
	}
	return action, banTTL, attempts, nil
}

func reputationField(s *requestSnapshot) string {
	fp := fingerprint(s.ip, s.ua, s.lang)
	return s.ip + "|" + fp
}

func fingerprint(ip string, ua string, lang string) string {
	raw := strings.ToLower(strings.TrimSpace(ip)) + "|" +
		strings.ToLower(strings.TrimSpace(ua)) + "|" +
		strings.ToLower(strings.TrimSpace(lang))
	h := hash64(raw)
	return strconv.FormatUint(h, 16)
}
