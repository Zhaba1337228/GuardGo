package load_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Zhaba1337228/GuardGo"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func BenchmarkEngineCheckSerial(b *testing.B) {
	mr := miniredis.RunT(b)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 1000000000,
		Window:      time.Second,
		FailOpen:    true,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer e.Close()

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = e.Check(ctx, req)
	}
}

func BenchmarkEngineCheckParallel(b *testing.B) {
	mr := miniredis.RunT(b)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 1000000000,
		Window:      time.Second,
		FailOpen:    true,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer e.Close()

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req, _ := http.NewRequest("GET", "http://example.com/test", nil)
		req.RemoteAddr = "127.0.0.1:23456"
		for pb.Next() {
			_, _, _ = e.Check(ctx, req)
		}
	})
}

func BenchmarkEngineCheckRateLimitedParallel(b *testing.B) {
	mr := miniredis.RunT(b)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 1,
		Window:      time.Second,
		FailOpen:    false,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer e.Close()

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req, _ := http.NewRequest("GET", "http://example.com/limited", nil)
		req.RemoteAddr = "127.0.0.1:34567"
		for pb.Next() {
			_, _, _ = e.Check(ctx, req)
		}
	})
}
