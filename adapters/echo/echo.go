package echo

import (
	"net/http"

	"guardgo"

	"github.com/labstack/echo/v4"
)

// Guard is an Echo middleware adapter for guardgo.Engine.
func Guard(cfg guardgo.MiddlewareConfig) echo.MiddlewareFunc {
	h, _, err := GuardWithEngine(cfg)
	if err != nil {
		if cfg.OnError != nil {
			cfg.OnError(err)
		}
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error { return next(c) }
		}
	}
	return h
}

// GuardWithEngine returns Echo middleware and the underlying Engine.
func GuardWithEngine(cfg guardgo.MiddlewareConfig) (echo.MiddlewareFunc, *guardgo.Engine, error) {
	engine, err := guardgo.NewEngine(cfg)
	if err != nil {
		return nil, nil, err
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			decision := engine.Process(c.Request().Context(), c.Request())
			guardgo.ApplyRateLimitHeaders(http.Header(c.Response().Header()), decision)
			if !decision.Allowed {
				return c.NoContent(decision.StatusCodeOr(engine.Config().DenyStatusCode))
			}
			return next(c)
		}
	}, engine, nil
}

// MustGuard is like GuardWithEngine but panics on construction error.
func MustGuard(cfg guardgo.MiddlewareConfig) (echo.MiddlewareFunc, *guardgo.Engine) {
	h, engine, err := GuardWithEngine(cfg)
	if err != nil {
		panic(err)
	}
	return h, engine
}
