package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Zhaba1337228/GuardGo"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSelfHealingReducesEffectiveLimitOnBurst(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	const baseLimit = 20
	engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: baseLimit,
		Window:      time.Second,
		SelfHealing: guardgo.SelfHealingConfig{
			Enabled:               true,
			Window:                time.Minute,
			MinSamples:            10,
			CleanTrafficThreshold: 0.9,
			SpikeFactor:           0.5,
			MaxSensitivity:        4.0,
			DecayStep:             0.1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	ctx := context.Background()

	// Build clean baseline with many different IPs.
	for i := 0; i < 12; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/baseline", nil)
		req.RemoteAddr = fmt.Sprintf("60.0.0.%d:1000", i+1)
		d := engine.Process(ctx, req)
		if !d.Allowed {
			t.Fatalf("baseline request should be allowed, got %+v", d)
		}
	}

	attackerReq, _ := http.NewRequest(http.MethodGet, "http://example.com/attack", nil)
	attackerReq.RemoteAddr = "70.7.7.7:7777"

	minObservedLimit := baseLimit
	for i := 0; i < 25; i++ {
		d := engine.Process(ctx, attackerReq)
		if d.Limit < minObservedLimit {
			minObservedLimit = d.Limit
		}
	}

	if minObservedLimit >= baseLimit {
		t.Fatalf("expected self-healing to reduce effective limit below %d, got min %d", baseLimit, minObservedLimit)
	}
}
