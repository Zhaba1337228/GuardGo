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

## DFA Signatures and Hot Swap
- `pkg/dfa/matcher.go`: deterministic finite automaton matcher (`O(N)` scan).
- `ruleset.go`: YAML/JSON ruleset loader and hot-swappable `RulesetManager`.
- Flow: rules file -> compile to DFA -> background risk scoring.

## Behavioral Engine
- `behavior_tracker.go`: per-IP query/header diversity tracking.
- `rule.go`: classic `Rule` + score-based `Evaluator` interfaces.
- `internal/redislua/reputation.go`: score->penalty->blacklist decision and dynamic backoff bans in Lua.
- `engine.go`: scoring pipeline, reputation hash updates, penalty box activation, strict checks.

## Agent Sidecar
- `cmd/guardgo-agent/main.go`: Redis scanner + Prometheus `/metrics` endpoint for sidecar mode.
- `cmd/guardgo-cli/main.go`: realtime terminal dashboard (risk top, active blocks, DFA stats).

## Framework Adapters
- `adapters/gin/gin.go`: Gin adapter built on top of `guardgo.Engine`.
- `adapters/echo/echo.go`: Echo adapter built on top of `guardgo.Engine`.
- `adapters/fiber/fiber.go`: Fiber adapter built on top of `guardgo.Engine`.

## Rules
- `rules/basic.go`: reusable built-in rules for URL, query, and headers.

## Testing and Load Validation
- `tests/integration/*`: integration and behavior tests for pipeline, penalty, tracing, self-healing, and frameworks.
- `tests/load/engine_benchmark_test.go`: serial and parallel load benchmarks.
- `tests/chaos/*`: Redis outage chaos tests via `testcontainers-go` (run with `-tags chaos`).
