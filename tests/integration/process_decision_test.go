package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"guardgo"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestProcessDecisionFields(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 2,
		Window:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/process", nil)
	req.RemoteAddr = "30.1.1.1:1111"
	ctx := context.Background()

	d1 := engine.Process(ctx, req)
	if !d1.Allowed || d1.Reason != guardgo.ReasonAllowed || d1.Limit != 2 || d1.Counter != 1 || d1.Remaining != 1 {
		t.Fatalf("unexpected first decision: %+v", d1)
	}
	if time.Until(d1.ResetAt) <= 0 {
		t.Fatalf("expected future reset time, got %v", d1.ResetAt)
	}

	d2 := engine.Process(ctx, req)
	if !d2.Allowed || d2.Counter != 2 || d2.Remaining != 0 {
		t.Fatalf("unexpected second decision: %+v", d2)
	}

	d3 := engine.Process(ctx, req)
	if d3.Allowed || d3.Reason != guardgo.ReasonRateLimited || d3.Counter < 3 || d3.Remaining != 0 {
		t.Fatalf("unexpected rate-limited decision: %+v", d3)
	}
}
