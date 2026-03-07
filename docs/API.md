# Public API Guide

## Usage Levels

- `Level 1 (Basic)`: one-line startup, default behavior.
- `Level 2 (Intermediate)`: tuned limits, framework wrappers, presets.
- `Level 3 (Advanced)`: reputation/evaluators, ruleset hot reload, observability.

## Engine Constructors

- `New(cfg MiddlewareConfig) *Engine`
  - Panic-on-error constructor for shortest integration path.
- `NewWithError(cfg MiddlewareConfig) (*Engine, error)`
  - Explicit error handling constructor.
- `NewDefault() *Engine`
  - One-line default engine (`127.0.0.1:6379`, `1000 req/s`).
- `NewWithRedisAddr(redisAddr string) *Engine`
  - One-line constructor with custom Redis address.

### Examples

`Level 1 (Basic)`:

```go
engine := guardgo.NewDefault()
defer engine.Close()
```

`Level 2 (Intermediate)`:

```go
engine := guardgo.NewWithRedisAddr("127.0.0.1:6379")
defer engine.Close()
```

`Level 3 (Advanced)`:

```go
cfg := guardgo.DefaultConfig()
cfg.FailOpen = true
cfg.Bloom.Enabled = true
cfg.Reputation.Enabled = true
engine := guardgo.New(cfg)
defer engine.Close()
```

## Config Helpers

- `DefaultConfig() MiddlewareConfig`
- `NewConfig(redisClient *redis.Client, maxRequests int, window time.Duration) MiddlewareConfig`
- `NewConfigFromRedisAddr(redisAddr string, maxRequests int, window time.Duration) MiddlewareConfig`

### Examples

`Level 1 (Basic)`:

```go
cfg := guardgo.DefaultConfig()
```

`Level 2 (Intermediate)`:

```go
cfg := guardgo.NewConfigFromRedisAddr("127.0.0.1:6379", 300, time.Second)
cfg.FailOpen = true
```

`Level 3 (Advanced)`:

```go
cfg := guardgo.NewConfig(redisClient, 500, time.Second)
cfg.Bloom.Enabled = true
cfg.Tracing.Enabled = true
cfg.Reputation.Enabled = true
cfg.Reputation.WarningLevel = 60
cfg.Reputation.Threshold = 100
```

## HTTP Integration

- `(*Engine).Middleware(next http.Handler) http.Handler`
- `Guard(cfg MiddlewareConfig) (func(http.Handler) http.Handler, error)`
- `MustGuard(cfg MiddlewareConfig) func(http.Handler) http.Handler`

### Examples

`Level 1 (Basic)`:

```go
engine := guardgo.NewDefault()
http.ListenAndServe(":8080", engine.Middleware(mux))
```

`Level 2 (Intermediate)`:

```go
mw, err := guardgo.Guard(cfg)
if err != nil { panic(err) }
http.ListenAndServe(":8080", mw(mux))
```

`Level 3 (Advanced)`:

```go
cfg.DenyStatusCode = http.StatusForbidden
mw := guardgo.MustGuard(cfg)
http.ListenAndServe(":8080", mw(mux))
```

## Framework Wrappers

- `Gin(engine *Engine) gin.HandlerFunc`
- `Echo(engine *Engine) echo.MiddlewareFunc`
- `Fiber(engine *Engine) fiber.Handler`

### Examples

`Level 1 (Basic)`:

```go
engine := guardgo.NewDefault()
router.Use(guardgo.Gin(engine))
```

`Level 2 (Intermediate)`:

```go
engine := guardgo.New(cfg)
e.Use(guardgo.Echo(engine))
```

`Level 3 (Advanced)`:

```go
engine := guardgo.New(cfg)
app.Use(guardgo.Fiber(engine))
```

## Decision API

- `(*Engine).Process(ctx context.Context, r *http.Request) Decision`
- `(*Engine).Check(ctx context.Context, r *http.Request) (allowed bool, reason Reason, counter int64)`

`Decision` includes:
- `Allowed`
- `Reason`
- `Counter`
- `Limit`
- `Remaining`
- `ResetAt`
- `StatusCode`

### Examples

`Level 1 (Basic)`:

```go
allowed, reason, _ := engine.Check(ctx, req)
if !allowed { _ = reason }
```

`Level 2 (Intermediate)`:

```go
d := engine.Process(ctx, req)
if !d.Allowed {
  w.WriteHeader(d.StatusCodeOr(http.StatusTooManyRequests))
}
```

`Level 3 (Advanced)`:

```go
d := engine.Process(ctx, req)
guardgo.ApplyRateLimitHeaders(w.Header(), d)
if !d.Allowed {
  w.WriteHeader(d.StatusCodeOr(cfg.DenyStatusCode))
  return
}
```

## Rules and Presets

- `Rule` (`Evaluate(*http.Request) int`) for static checks.
- `Evaluator` (`CalculateScore(*http.Request) float64`) for reputation scoring.
- `rules.DefaultSecurityPresets() []guardgo.SignatureRule`
- `rules.ApplyDefaultSecurityPresets(cfg *guardgo.MiddlewareConfig)`

### Examples

`Level 1 (Basic)`:

```go
rules.ApplyDefaultSecurityPresets(&cfg)
```

`Level 2 (Intermediate)`:

```go
cfg.Rules = []guardgo.Rule{
  rules.PathContains{Tokens: []string{"/wp-admin"}, Weight: 2},
}
```

`Level 3 (Advanced)`:

```go
type LoginEvaluator struct{}
func (LoginEvaluator) CalculateScore(r *http.Request) float64 { return 12 }
cfg.Evaluators = []guardgo.Evaluator{LoginEvaluator{}}
cfg.Reputation.Enabled = true
```

## Rulesets and Hot Reload

- `NewRulesetManager(path string) (*RulesetManager, error)`
- `(*RulesetManager).Reload() error`
- `LoadRulesetFile(path string) (*CompiledRuleset, error)`
- `CompileRuleset(rules []SignatureRule) *CompiledRuleset`

### Examples

`Level 1 (Basic)`:

```go
cfg.RulesetFile = "./rules.yaml"
```

`Level 2 (Intermediate)`:

```go
manager, err := guardgo.NewRulesetManager("./rules.yaml")
if err != nil { panic(err) }
cfg.DynamicRules = manager
```

`Level 3 (Advanced)`:

```go
compiled, err := guardgo.LoadRulesetFile("./rules.yaml")
if err != nil { panic(err) }
cfg.CompiledRules = compiled
cfg.AutoReloadOnSIGHUP = true
```
