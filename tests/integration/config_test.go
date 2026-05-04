package integration_test

import (
	"testing"

	"github.com/Zhaba1337228/GuardGo"
)

func TestDefaultConfigProvidesReadyRedisClient(t *testing.T) {
	cfg := guardgo.DefaultConfig()
	if cfg.Redis == nil {
		t.Fatalf("expected redis client to be initialized in DefaultConfig")
	}
	if cfg.MaxRequests != 1000 {
		t.Fatalf("unexpected default max requests: got=%d want=1000", cfg.MaxRequests)
	}
}

func TestNewEngineWithEmptyConfigInitializesRedis(t *testing.T) {
	engine, err := guardgo.NewWithError(guardgo.MiddlewareConfig{})
	if err != nil {
		t.Fatalf("expected empty config to auto-initialize, got error: %v", err)
	}
	defer engine.Close()
	if engine.Config().Redis == nil {
		t.Fatalf("expected redis client to be auto-initialized")
	}
}
