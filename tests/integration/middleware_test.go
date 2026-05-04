package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	guardgo "github.com/Zhaba1337228/GuardGo"
)

func TestHTTPMiddlewareBehavior(t *testing.T) {
	t.Run("allows_and_calls_next_handler", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 10,
			Window:      time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		nextCalled := false
		handler := engine.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusNoContent)
		}))

		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		req.RemoteAddr = "20.1.1.1:3000"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent || !nextCalled {
			t.Fatalf("expected next handler to be called with 204, got code=%d next_called=%v", rr.Code, nextCalled)
		}
		if rr.Header().Get("X-RateLimit-Limit") == "" ||
			rr.Header().Get("X-RateLimit-Remaining") == "" ||
			rr.Header().Get("X-RateLimit-Reset") == "" {
			t.Fatalf("expected rate-limit headers to be present on allowed response")
		}
	})

	t.Run("blocks_with_custom_deny_status_code", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:          rdb,
			MaxRequests:    1,
			Window:         time.Second,
			DenyStatusCode: http.StatusForbidden,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		handler := engine.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))

		req := httptest.NewRequest(http.MethodGet, "/block", nil)
		req.RemoteAddr = "20.2.2.2:3000"

		first := httptest.NewRecorder()
		handler.ServeHTTP(first, req)
		if first.Code != http.StatusNoContent {
			t.Fatalf("expected first request pass-through 204, got %d", first.Code)
		}

		second := httptest.NewRecorder()
		handler.ServeHTTP(second, req)
		if second.Code != http.StatusForbidden {
			t.Fatalf("expected deny status %d, got %d", http.StatusForbidden, second.Code)
		}
		if second.Header().Get("X-RateLimit-Limit") == "" ||
			second.Header().Get("X-RateLimit-Remaining") == "" ||
			second.Header().Get("X-RateLimit-Reset") == "" {
			t.Fatalf("expected rate-limit headers to be present on denied response")
		}
	})
}
