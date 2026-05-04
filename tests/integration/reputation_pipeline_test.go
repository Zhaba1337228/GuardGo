package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Zhaba1337228/GuardGo"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type constEvaluator struct {
	score float64
}

func (e constEvaluator) CalculateScore(r *http.Request) float64 {
	_ = r
	return e.score
}

func TestReputationPipelineWithEvaluator(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 100,
		Window:      time.Second,
		Evaluators: []guardgo.Evaluator{
			constEvaluator{score: 80},
		},
		Reputation: guardgo.ReputationConfig{
			Enabled:      true,
			WarningLevel: 60,
			Threshold:    1000,
			PenaltyTTL:   time.Minute,
		},
		Behavioral: guardgo.BehavioralConfig{
			Enabled:          true,
			RequireUserAgent: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/reputation", nil)
	req.RemoteAddr = "90.9.9.9:9000"
	ctx := context.Background()

	_ = engine.Process(ctx, req)

	deadline := time.Now().Add(800 * time.Millisecond)
	for {
		d := engine.Process(ctx, req)
		if !d.Allowed && d.Reason == guardgo.ReasonPenaltyForbidden {
			if d.StatusCodeOr(http.StatusTooManyRequests) != http.StatusForbidden {
				t.Fatalf("expected 403 in penalty mode, got %d", d.StatusCodeOr(http.StatusTooManyRequests))
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected penalty forbidden decision before timeout")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
