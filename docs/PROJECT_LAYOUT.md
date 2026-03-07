# Project Layout

## Core
- `config.go`: public configuration models and defaults.
- `engine.go`: decision engine (critical path + background scoring).
- `rule.go`: `Rule` and `Evaluator` interfaces.
- `middleware_http.go`: `net/http` middleware and standard rate-limit headers.
- `frameworks.go`: unified wrappers for Gin/Echo/Fiber.

## Internal
- `internal/redislua/script.go`: atomic rate-limit + blacklist Lua.
- `internal/redislua/reputation.go`: reputation score + penalty/blacklist Lua.
- `internal/blacklist/cache.go`: in-memory TTL LRU caches.

## Algorithm Packages
- `pkg/bloom`: probabilistic fast path.
- `pkg/dfa`: deterministic finite automaton matcher for signatures.

## Operational Components
- `cmd/guardgo-agent`: sidecar agent exposing Prometheus metrics.
- `ruleset.go`: JSON/YAML ruleset loading and hot reload manager.

## Tests
- `tests/integration`: behavior/integration scenarios using miniredis.
- `tests/load`: benchmark and load-oriented tests.
- `tests/chaos`: optional chaos tests with `testcontainers-go` (`-tags chaos`).
- `pkg/*` and `rules/*`: package-level unit tests.

## Examples
- `examples/nethttp-basic`: minimal net/http integration.
- `examples/gin-basic`: minimal Gin integration.
