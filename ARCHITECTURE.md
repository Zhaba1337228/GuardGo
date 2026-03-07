# GuardGo Architecture

## Public API (`guardgo`)
- `config.go`: middleware and adaptive-ban configuration.
- `engine.go`: critical-path request checks and background rule pipeline.
- `helpers.go`: quick-start constructors and middleware helpers.
- `frameworks.go`: unified plugin-style wrappers (`guardgo.Gin/Echo/Fiber`).
- `middleware_http.go`: `net/http` middleware adapter.
- `rule.go`: `Rule` abstraction.
- `metrics.go`: metrics interface.
- `stats.go`: generic dependency-injected telemetry interface (`StatsCollector`).
- `tracing.go`: OpenTelemetry span creation and decision/redis annotations.
- `self_healing.go`: adaptive behavior model for dynamic effective limits.

## Internal Implementation
- `internal/redislua/script.go`: atomic Redis Lua script for blacklist + rate-limit.
- `internal/blacklist/cache.go`: in-process blacklist LRU cache with TTL entries.

## Probabilistic Fast Path
- `pkg/bloom/filter.go`: lock-free in-memory bloom filter for blacklist candidates.
- Flow: cache blacklist -> bloom negative short-circuit -> Redis Lua fallback.

## Agent Sidecar
- `cmd/guardgo-agent/main.go`: Redis scanner + Prometheus `/metrics` endpoint for sidecar mode.

## Framework Adapters
- `adapters/gin/gin.go`: Gin adapter built on top of `guardgo.Engine`.
- `adapters/echo/echo.go`: Echo adapter built on top of `guardgo.Engine`.
- `adapters/fiber/fiber.go`: Fiber adapter built on top of `guardgo.Engine`.

## Rules
- `rules/basic.go`: reusable built-in rules for URL, query, and headers.

## Testing and Load Validation
- `tests/load/engine_test.go`: functional and stress tests.
- `tests/load/engine_behavior_test.go`: behavior tests for fail-open/close, adaptive ban, bloom, stats.
- `tests/load/engine_benchmark_test.go`: serial and parallel load benchmarks.
- `tests/load/tracing_test.go`: OpenTelemetry span assertions.
- `tests/load/self_healing_test.go`: adaptive limit assertions under burst traffic.
