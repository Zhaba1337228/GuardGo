package guardgo

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// New returns a new Engine instance and panics if construction fails.
// This enables 1-line DX: guard := guardgo.New(guardgo.DefaultConfig()).
func New(cfg MiddlewareConfig) *Engine {
	engine, err := NewEngine(cfg)
	if err != nil {
		panic(err)
	}
	return engine
}

// NewWithError returns a new Engine instance with explicit error handling.
func NewWithError(cfg MiddlewareConfig) (*Engine, error) {
	return NewEngine(cfg)
}

// MustNew returns a new Engine instance or panics.
func MustNew(cfg MiddlewareConfig) *Engine {
	return New(cfg)
}

// NewConfig returns a minimal ready-to-use configuration.
func NewConfig(redisClient *redis.Client, maxRequests int, window time.Duration) MiddlewareConfig {
	return MiddlewareConfig{
		Redis:       redisClient,
		MaxRequests: maxRequests,
		Window:      window,
	}
}

// Guard returns net/http middleware from config.
func Guard(cfg MiddlewareConfig) (func(http.Handler) http.Handler, error) {
	engine, err := NewWithError(cfg)
	if err != nil {
		return nil, err
	}
	return engine.Middleware, nil
}

// MustGuard returns net/http middleware from config or panics.
func MustGuard(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	middleware, err := Guard(cfg)
	if err != nil {
		panic(err)
	}
	return middleware
}
