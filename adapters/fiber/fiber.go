package fiber

import (
	"context"
	"net/http"
	"net/url"

	guardgo "github.com/Zhaba1337228/GuardGo"

	"github.com/gofiber/fiber/v2"
)

// Guard is a Fiber middleware adapter for guardgo.Engine.
func Guard(cfg guardgo.MiddlewareConfig) fiber.Handler {
	h, _, err := GuardWithEngine(cfg)
	if err != nil {
		if cfg.OnError != nil {
			cfg.OnError(err)
		}
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	return h
}

// GuardWithEngine returns Fiber middleware and the underlying Engine.
func GuardWithEngine(cfg guardgo.MiddlewareConfig) (fiber.Handler, *guardgo.Engine, error) {
	engine, err := guardgo.NewEngine(cfg)
	if err != nil {
		return nil, nil, err
	}
	return func(c *fiber.Ctx) error {
		req := &http.Request{
			Method:     c.Method(),
			Host:       c.Hostname(),
			RemoteAddr: c.IP(),
			Header:     make(http.Header),
			URL: &url.URL{
				Path:     c.Path(),
				RawQuery: string(c.Request().URI().QueryString()),
			},
		}
		c.Request().Header.VisitAll(func(k, v []byte) {
			req.Header.Add(string(k), string(v))
		})

		decision := engine.Process(context.Background(), req)
		for headerKey, values := range makeRateHeaders(decision) {
			for _, value := range values {
				c.Append(headerKey, value)
			}
		}
		if !decision.Allowed {
			return c.SendStatus(decision.StatusCodeOr(engine.Config().DenyStatusCode))
		}
		return c.Next()
	}, engine, nil
}

// MustGuard is like GuardWithEngine but panics on construction error.
func MustGuard(cfg guardgo.MiddlewareConfig) (fiber.Handler, *guardgo.Engine) {
	h, engine, err := GuardWithEngine(cfg)
	if err != nil {
		panic(err)
	}
	return h, engine
}

func makeRateHeaders(decision guardgo.Decision) http.Header {
	h := make(http.Header, 4)
	guardgo.ApplyRateLimitHeaders(h, decision)
	return h
}
