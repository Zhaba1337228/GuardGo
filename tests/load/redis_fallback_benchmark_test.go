package load_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	guardgo "github.com/Zhaba1337228/GuardGo"
)

func BenchmarkRedisFallbackRealRedis(b *testing.B) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		b.Skip("set REDIS_ADDR to run real Redis fallback benchmark")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		PoolSize:     128,
		MinIdleConns: 32,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		b.Fatalf("redis ping failed: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 1_000_000_000,
		Window:      time.Second,
		FailOpen:    false,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer engine.Close()

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/fallback", nil)
	req.RemoteAddr = "198.51.100.5:12345"

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = engine.Process(ctx, req)
		}
	})
}
