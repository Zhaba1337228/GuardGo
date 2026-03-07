package guardgo

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
)

// Gin returns a Gin middleware based on the provided Engine.
func Gin(engine *Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision := engine.Process(c.Request.Context(), c.Request)
		ApplyRateLimitHeaders(c.Writer.Header(), decision)
		if !decision.Allowed {
			c.AbortWithStatus(engine.Config().DenyStatusCode)
			return
		}
		c.Next()
	}
}

// Echo returns an Echo middleware based on the provided Engine.
func Echo(engine *Engine) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			decision := engine.Process(c.Request().Context(), c.Request())
			ApplyRateLimitHeaders(http.Header(c.Response().Header()), decision)
			if !decision.Allowed {
				return c.NoContent(engine.Config().DenyStatusCode)
			}
			return next(c)
		}
	}
}

// Fiber returns a Fiber middleware based on the provided Engine.
func Fiber(engine *Engine) fiber.Handler {
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
		rateHeaders := make(http.Header, 4)
		ApplyRateLimitHeaders(rateHeaders, decision)
		for headerKey, values := range rateHeaders {
			for _, value := range values {
				c.Append(headerKey, value)
			}
		}
		if !decision.Allowed {
			return c.SendStatus(engine.Config().DenyStatusCode)
		}
		return c.Next()
	}
}
