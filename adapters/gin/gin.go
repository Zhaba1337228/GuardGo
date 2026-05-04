package gin

import (
	guardgo "github.com/Zhaba1337228/GuardGo"

	"github.com/gin-gonic/gin"
)

// Guard is a Gin middleware adapter for guardgo.Engine.
//
// If Engine construction fails, Guard returns a fail-open middleware and reports
// the error via cfg.OnError (if configured).
//
// Note: When using Gin behind proxies, configure Gin's trusted proxies and/or
// set guardgo.MiddlewareConfig.IPFunc to use X-Forwarded-For correctly.
func Guard(cfg guardgo.MiddlewareConfig) gin.HandlerFunc {
	h, _, err := GuardWithEngine(cfg)
	if err != nil {
		if cfg.OnError != nil {
			cfg.OnError(err)
		}
		return func(c *gin.Context) { c.Next() }
	}
	return h
}

// GuardWithEngine returns the Gin middleware and the underlying Engine.
func GuardWithEngine(cfg guardgo.MiddlewareConfig) (gin.HandlerFunc, *guardgo.Engine, error) {
	e, err := guardgo.NewEngine(cfg)
	if err != nil {
		return nil, nil, err
	}
	return func(c *gin.Context) {
		decision := e.Process(c.Request.Context(), c.Request)
		guardgo.ApplyRateLimitHeaders(c.Writer.Header(), decision)
		if !decision.Allowed {
			c.AbortWithStatus(decision.StatusCodeOr(e.Config().DenyStatusCode))
			return
		}
		c.Next()
	}, e, nil
}

// MustGuard is like GuardWithEngine but panics on construction error.
func MustGuard(cfg guardgo.MiddlewareConfig) (gin.HandlerFunc, *guardgo.Engine) {
	h, e, err := GuardWithEngine(cfg)
	if err != nil {
		panic(err)
	}
	return h, e
}
