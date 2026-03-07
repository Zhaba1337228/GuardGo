package guardgo

import (
	"net/http"
	"strconv"
	"time"
)

// Middleware returns a net/http middleware.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decision := e.Process(r.Context(), r)
		ApplyRateLimitHeaders(w.Header(), decision)
		if !decision.Allowed {
			w.WriteHeader(e.cfg.DenyStatusCode)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GuardHTTP is a convenience helper to construct an Engine and return a net/http middleware.
func GuardHTTP(cfg MiddlewareConfig) (func(http.Handler) http.Handler, *Engine, error) {
	e, err := NewEngine(cfg)
	if err != nil {
		return nil, nil, err
	}
	return e.Middleware, e, nil
}

// ApplyRateLimitHeaders writes standard rate-limit headers for client visibility.
func ApplyRateLimitHeaders(h http.Header, d Decision) {
	h.Set("X-RateLimit-Limit", strconv.Itoa(d.Limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(d.Remaining))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(d.ResetAt.Unix(), 10))
	h.Set("X-RateLimit-Reset-At", d.ResetAt.UTC().Format(time.RFC3339))
}
