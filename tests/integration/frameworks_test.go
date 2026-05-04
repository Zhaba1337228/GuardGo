package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	guardgo "github.com/Zhaba1337228/GuardGo"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func TestFrameworkWrappers(t *testing.T) {
	t.Run("gin_wrapper_sets_headers_and_blocks", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(guardgo.Gin(engine))
		router.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
		req1.RemoteAddr = "40.1.1.1:1000"
		rr1 := httptest.NewRecorder()
		router.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusNoContent {
			t.Fatalf("expected first gin request to pass, got %d", rr1.Code)
		}
		assertRateHeaders(t, rr1.Header())

		req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
		req2.RemoteAddr = "40.1.1.1:1000"
		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusTooManyRequests {
			t.Fatalf("expected second gin request to be blocked, got %d", rr2.Code)
		}
		assertRateHeaders(t, rr2.Header())
	})

	t.Run("echo_wrapper_sets_headers_and_blocks", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		e := echo.New()
		e.Use(guardgo.Echo(engine))
		e.GET("/x", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })

		req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
		req1.RemoteAddr = "40.2.2.2:1000"
		rec1 := httptest.NewRecorder()
		e.ServeHTTP(rec1, req1)
		if rec1.Code != http.StatusNoContent {
			t.Fatalf("expected first echo request to pass, got %d", rec1.Code)
		}
		assertRateHeaders(t, rec1.Header())

		req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
		req2.RemoteAddr = "40.2.2.2:1000"
		rec2 := httptest.NewRecorder()
		e.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusTooManyRequests {
			t.Fatalf("expected second echo request to be blocked, got %d", rec2.Code)
		}
		assertRateHeaders(t, rec2.Header())
	})

	t.Run("fiber_wrapper_sets_headers_and_blocks", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
			Redis:       rdb,
			MaxRequests: 1,
			Window:      time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer engine.Close()

		app := fiber.New()
		app.Use(guardgo.Fiber(engine))
		app.Get("/x", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
		req1.RemoteAddr = "40.3.3.3:1000"
		resp1, err := app.Test(req1, -1)
		if err != nil {
			t.Fatal(err)
		}
		defer resp1.Body.Close()
		if resp1.StatusCode != http.StatusNoContent {
			t.Fatalf("expected first fiber request to pass, got %d", resp1.StatusCode)
		}
		assertRateHeaders(t, resp1.Header)

		req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
		req2.RemoteAddr = "40.3.3.3:1000"
		resp2, err := app.Test(req2, -1)
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("expected second fiber request to be blocked, got %d", resp2.StatusCode)
		}
		assertRateHeaders(t, resp2.Header)
	})
}

func assertRateHeaders(t *testing.T, h http.Header) {
	t.Helper()
	if h.Get("X-RateLimit-Limit") == "" {
		t.Fatalf("missing X-RateLimit-Limit header")
	}
	if h.Get("X-RateLimit-Remaining") == "" {
		t.Fatalf("missing X-RateLimit-Remaining header")
	}
	if h.Get("X-RateLimit-Reset") == "" {
		t.Fatalf("missing X-RateLimit-Reset header")
	}
	if h.Get("X-RateLimit-Reset-At") == "" {
		t.Fatalf("missing X-RateLimit-Reset-At header")
	}
}
