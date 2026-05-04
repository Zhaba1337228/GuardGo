package guardgo

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestApplyRateLimitHeaders(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	d := Decision{Limit: 100, Remaining: 42, ResetAt: now}

	h := http.Header{}
	ApplyRateLimitHeaders(h, d)

	if got := h.Get("X-RateLimit-Limit"); got != "100" {
		t.Fatalf("limit header: want 100, got %q", got)
	}
	if got := h.Get("X-RateLimit-Remaining"); got != "42" {
		t.Fatalf("remaining header: want 42, got %q", got)
	}
	if got := h.Get("X-RateLimit-Reset"); got != strconv.FormatInt(now.Unix(), 10) {
		t.Fatalf("reset header: want unix %d, got %q", now.Unix(), got)
	}
	if got := h.Get("X-RateLimit-Reset-At"); got != now.Format(time.RFC3339) {
		t.Fatalf("reset-at header: want %s, got %q", now.Format(time.RFC3339), got)
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(nil, 250, 2*time.Second)
	if cfg.MaxRequests != 250 {
		t.Fatalf("MaxRequests: want 250, got %d", cfg.MaxRequests)
	}
	if cfg.Window != 2*time.Second {
		t.Fatalf("Window: want 2s, got %s", cfg.Window)
	}
}

func TestNewConfigFromRedisAddr(t *testing.T) {
	t.Parallel()
	cfg := NewConfigFromRedisAddr("127.0.0.1:6379", 50, time.Second)
	if cfg.Redis == nil {
		t.Fatal("Redis client must be set")
	}
	if cfg.MaxRequests != 50 || cfg.Window != time.Second {
		t.Fatalf("config fields not propagated: %+v", cfg)
	}
}

func TestNewWithRedisAddr_FallsBackToDefault(t *testing.T) {
	t.Parallel()
	e := NewWithRedisAddr("")
	t.Cleanup(e.Close)
	if e.cfg.Redis == nil {
		t.Fatal("Redis client must default when address empty")
	}
}

func TestMustGuard_ReturnsMiddleware(t *testing.T) {
	t.Parallel()
	mw := MustGuard(MiddlewareConfig{
		MaxRequests: 100,
		Window:      time.Second,
	})
	if mw == nil {
		t.Fatal("MustGuard returned nil middleware")
	}
}

func TestGuard_ReturnsMiddleware(t *testing.T) {
	t.Parallel()
	mw, err := Guard(MiddlewareConfig{
		MaxRequests: 1000,
		Window:      time.Second,
	})
	if err != nil {
		t.Fatalf("Guard: %v", err)
	}
	if mw == nil {
		t.Fatal("Guard returned nil middleware")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	_ = rec
	_ = req
	_ = called
}

func TestGuardHTTP_ReturnsEngine(t *testing.T) {
	t.Parallel()
	mw, engine, err := GuardHTTP(MiddlewareConfig{
		MaxRequests: 1,
		Window:      time.Second,
	})
	if err != nil {
		t.Fatalf("GuardHTTP: %v", err)
	}
	if engine == nil || mw == nil {
		t.Fatal("GuardHTTP returned nil")
	}
	t.Cleanup(engine.Close)
}
