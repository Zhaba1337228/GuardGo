package integration_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	guardgo "github.com/Zhaba1337228/GuardGo"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestPenaltyBoxForbiddenAndBan(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "rules.yaml")
	rulesYAML := []byte("rules:\n  - name: SQLi\n    match: query\n    pattern: \"(?i)(union|select)\"\n    weight: 60\n")
	if err := os.WriteFile(rulesPath, rulesYAML, 0o600); err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 100,
		Window:      time.Second,
		RulesetFile: rulesPath,
		Behavioral: guardgo.BehavioralConfig{
			Enabled:          true,
			ScoreThreshold:   50,
			RiskWindow:       time.Minute,
			PenaltyTTL:       time.Minute,
			PenaltyBanTTL:    24 * time.Hour,
			RequireUserAgent: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/login?q=union+select", nil)
	req.RemoteAddr = "80.8.8.8:8080"
	ctx := context.Background()

	_ = engine.Process(ctx, req)

	deadline := time.Now().Add(800 * time.Millisecond)
	for {
		d := engine.Process(ctx, req)
		if !d.Allowed && d.Reason == guardgo.ReasonPenaltyForbidden {
			if d.StatusCodeOr(http.StatusTooManyRequests) != http.StatusForbidden {
				t.Fatalf("expected status 403 for penalty-forbidden, got %d", d.StatusCodeOr(http.StatusTooManyRequests))
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected request to be denied by penalty box before timeout")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// After penalty failure, IP is banned for PenaltyBanTTL.
	d2 := engine.Process(ctx, req)
	if d2.Allowed || d2.Reason != guardgo.ReasonBlacklisted {
		t.Fatalf("expected subsequent request to be blacklisted, got %+v", d2)
	}
}
