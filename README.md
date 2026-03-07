# GuardGo

GuardGo is a Redis-backed API protection library for Go focused on high-load APIs.

Core ideas:
- atomic critical path via Redis Lua
- in-memory fast path (`LRU` + `Bloom`)
- reputation scoring pipeline (rules + evaluators + behavioral entropy)
- dynamic penalty and blacklist backoff
- framework adapters and operational tooling (`agent`, `cli`)

## Install

```bash
go get guardgo
go get github.com/redis/go-redis/v9
```

## Quick Start (`net/http`)

```go
package main

import (
	"net/http"
	"time"

	"guardgo"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

	cfg := guardgo.NewConfig(rdb, 200, 1*time.Second)
	cfg.FailOpen = true
	cfg.Bloom.Enabled = true
	cfg.Reputation.Enabled = true
	cfg.Reputation.WarningLevel = 60
	cfg.Reputation.Threshold = 100

	engine := guardgo.New(cfg)
	defer engine.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_ = http.ListenAndServe(":8080", engine.Middleware(mux))
}
```

## Zero-Config Mode

```go
guard := guardgo.New(guardgo.DefaultConfig())
defer guard.Close()
```

Defaults:
- Redis: `127.0.0.1:6379`
- limit: `1000` req / `1s`

Integration shortcuts:

```go
engine := guardgo.NewDefault()
engine2 := guardgo.NewWithRedisAddr("10.0.0.12:6379")
cfg := guardgo.NewConfigFromRedisAddr("127.0.0.1:6379", 200, time.Second)
```

## Framework Adapters

Unified wrappers:

```go
router.Use(guardgo.Gin(engine))
e.Use(guardgo.Echo(engine))
app.Use(guardgo.Fiber(engine))
```

Legacy adapters:
- `guardgo/adapters/gin`
- `guardgo/adapters/echo`
- `guardgo/adapters/fiber`

## Security Model

GuardGo evaluates traffic using a score pipeline:
- static `Rule` checks
- score-based `Evaluator` checks
- DFA signature matching from YAML/JSON rulesets
- behavioral entropy signals (query/header diversity)
- fingerprinting (`IP + User-Agent + Accept-Language`)

Decisions:
- score `< WarningLevel`: normal flow
- score `>= WarningLevel`: penalty mode
- score `>= Threshold`: blacklist

## Dynamic Backoff

Reputation Lua applies escalating bans:
- attack #1: `BanLevel1` (default `1m`)
- attack #2: `BanLevel2` (default `10m`)
- attack #3+: `BanLevel3` (default `24h`)

## Hot Reload (Zero-Downtime)

If `RulesetFile` or `DynamicRules` is configured, GuardGo can hot-reload rules on POSIX `SIGHUP`:

```bash
kill -HUP <pid>
```

No process restart required.

## Presets

Built-in baseline signatures:

```go
rules.ApplyDefaultSecurityPresets(&cfg) // in-memory, no rules.yaml needed
```

Includes common SQLi/probing/traversal/scanner patterns.

## Observability

- OpenTelemetry spans (`guardgo.process`)
- generic `StatsCollector` interface
- Prometheus sidecar: `cmd/guardgo-agent`
- live terminal dashboard: `cmd/guardgo-cli`

Run tools:

```bash
go run ./cmd/guardgo-agent --redis-addr 127.0.0.1:6379 --prefix guardgo --listen :9090
go run ./cmd/guardgo-cli --redis-addr 127.0.0.1:6379 --prefix guardgo --ruleset ./rules.yaml
```

## Real Benchmark Results

Measured on:
- CPU: `12th Gen Intel(R) Core(TM) i5-12400`
- OS: `windows/amd64`
- Date: `2026-03-08`

| Case | Latency | Throughput (approx) | Allocations |
|---|---:|---:|---:|
| Clean Request (Bloom) | `11.8ns` | `~84M ops/s` | `0 B/op, 0 allocs/op` |
| DFA Match (100 rules) | `~454ns` | `~2.2M ops/s` | `24 B/op, 1 alloc/op` |
| Redis Fallback (miniredis parallel) | `205-225µs` | `~4.4K ops/s/core-equivalent` | `~206KB/op, 836 allocs/op` |

Commands used:

```bash
go test ./pkg/bloom -run ^$ -bench BenchmarkBloomCleanRequest -benchmem -benchtime=5s -count=5
go test ./pkg/dfa -run ^$ -bench BenchmarkDFA100RulesMatch -benchmem -benchtime=5s -count=5
go test ./tests/load -run ^$ -bench BenchmarkEngineCheckParallel -benchmem -benchtime=10s -count=5
```

For real network Redis fallback (recommended for production profiling), set `REDIS_ADDR` and run:

```bash
go test ./tests/load -run ^$ -bench BenchmarkRedisFallbackRealRedis -benchmem -benchtime=10s -count=5
```

## Testing

```bash
go test ./...
go test -race ./tests/integration -count=1
go test ./tests/load -bench . -benchmem -run ^$
go test -tags chaos ./tests/chaos -v -count=1
```

## Repository Docs

- Architecture: `docs/ARCHITECTURE.md`
- Project layout: `docs/PROJECT_LAYOUT.md`
- API methods: `docs/API.md`
- Examples: `examples/`
