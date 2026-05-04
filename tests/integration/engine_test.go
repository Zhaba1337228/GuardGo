package integration_test

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	guardgo "github.com/Zhaba1337228/GuardGo"
)

func TestEngineRateLimit(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 2,
		Window:      50 * time.Millisecond,
		FailOpen:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	ctx := context.Background()

	if ok, reason, _ := e.Check(ctx, req); !ok || reason != guardgo.ReasonAllowed {
		t.Fatalf("1st: ok=%v reason=%v", ok, reason)
	}
	if ok, reason, _ := e.Check(ctx, req); !ok || reason != guardgo.ReasonAllowed {
		t.Fatalf("2nd: ok=%v reason=%v", ok, reason)
	}
	if ok, reason, _ := e.Check(ctx, req); ok || reason != guardgo.ReasonRateLimited {
		t.Fatalf("3rd: ok=%v reason=%v", ok, reason)
	}
}

func TestEngineStressConcurrent(t *testing.T) {
	const goroutines = 32
	const requestsPerGoroutine = 300

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: goroutines * requestsPerGoroutine * 2,
		Window:      time.Second,
		FailOpen:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	ctx := context.Background()
	var allowed int64
	var denied int64
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for workerID := 0; workerID < goroutines; workerID++ {
		id := workerID
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "http://example.com/stress", nil)
			req.RemoteAddr = "127.0.0.1:10000"
			for i := 0; i < requestsPerGoroutine; i++ {
				// Vary URL while preserving same limiter key (IP default).
				req.URL.Path = "/stress/" + string(rune('a'+(id%26)))
				ok, _, _ := e.Check(ctx, req)
				if ok {
					atomic.AddInt64(&allowed, 1)
				} else {
					atomic.AddInt64(&denied, 1)
				}
			}
		}()
	}
	wg.Wait()

	total := int64(goroutines * requestsPerGoroutine)
	if got := allowed + denied; got != total {
		t.Fatalf("total mismatch: got=%d want=%d", got, total)
	}
	if denied != 0 {
		t.Fatalf("unexpected denied requests: %d", denied)
	}
}
