# GuardGo

GuardGo is a Redis-backed API protection library for Go:
- atomic critical path via one Redis Lua call (`SISMEMBER + INCR + EXPIRE`)
- local blacklist LRU cache for fast-path blocks
- async rule engine for adaptive bans

## Install

```bash
go get github.com/redis/go-redis/v9
go get guardgo
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
cfg.AdaptiveBan.Enabled = true
cfg.Bloom.Enabled = true
cfg.Bloom.Bits = 1 << 20
cfg.Bloom.Hashes = 6
cfg.Tracing.Enabled = true
cfg.SelfHealing.Enabled = true

mw := guardgo.MustGuard(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

http.ListenAndServe(":8080", mw(mux))
}
```

## Zero-Config Mode

```go
guard := guardgo.New(guardgo.DefaultConfig())
defer guard.Close()
```

`DefaultConfig()` uses:
- Redis: `127.0.0.1:6379`
- rate limit: `1000` requests per `1s`

## Gin

```go
import (
	guardging "guardgo/adapters/gin"
)

router := gin.New()
router.Use(guardging.Guard(cfg))
```

Unified plugin-style API:

```go
guard := guardgo.New(guardgo.DefaultConfig())
router := gin.New()
router.Use(guardgo.Gin(guard))
```

## Echo

```go
import (
	guardecho "guardgo/adapters/echo"
)

e := echo.New()
e.Use(guardecho.Guard(cfg))
```

## Fiber

```go
import (
	guardfiber "guardgo/adapters/fiber"
)

app := fiber.New()
app.Use(guardfiber.Guard(cfg))
```

## Standard Rate-Limit Headers

GuardGo automatically writes these headers on both allowed and blocked responses:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset` (unix timestamp)
- `X-RateLimit-Reset-At` (RFC3339 UTC)

## Built-in Rules

```go
import "guardgo/rules"

cfg.Rules = []guardgo.Rule{
	rules.PathContains{Tokens: []string{"/wp-admin", "/xmlrpc.php"}, Weight: 2},
	rules.QueryContains{Tokens: []string{"union select", "or 1=1"}, Weight: 3},
	rules.HeaderContains{Name: "User-Agent", Tokens: []string{"sqlmap", "nikto"}, Weight: 5},
}
```

## Optional Stats Collector

```go
type myStats struct{}

func (myStats) Record(name string, value float64, labels map[string]string) {
	// forward to your telemetry backend
}

cfg.Stats = myStats{}
```

## OpenTelemetry Tracing

When `cfg.Tracing.Enabled = true`, GuardGo creates a span `guardgo.process` and adds attributes:
- `guardgo.allowed`
- `guardgo.reason`
- `guardgo.counter`
- `guardgo.limit`
- `guardgo.remaining`
- `guardgo.redis_ms`

## Self-Healing Adaptive Limits

Enable adaptive behavior model:

```go
cfg.SelfHealing.Enabled = true
cfg.SelfHealing.MinSamples = 500
cfg.SelfHealing.CleanTrafficThreshold = 0.9
cfg.SelfHealing.SpikeFactor = 3.0
```

When clean traffic dominates and one IP suddenly spikes, GuardGo lowers the effective limit for that period.

## DFA Signatures + Hot-Swap Rulesets

GuardGo compiles signature tokens into a DFA matcher (`O(N)` single-pass scan).

Load from file:

```go
cfg.RulesetFile = "./rules.yaml"
cfg.Behavioral.Enabled = true
```

Hot-swap programmatically:

```go
manager, _ := guardgo.NewRulesetManager("./rules.yaml")
cfg.DynamicRules = manager
// manager.Reload() on SIGHUP or admin endpoint
```

Example `rules.yaml`:

```yaml
rules:
  - name: "SQLi Injection"
    match: "query"
    pattern: "(?i)(union|select|drop|insert)"
    weight: 10
  - name: "Header Probe"
    match: "headers"
    pattern: "keep-alive"
    weight: 20
```

## Penalty Box

Enable behavioral score and strict mode:

```go
cfg.Behavioral.Enabled = true
cfg.Behavioral.ScoreThreshold = 100
cfg.Behavioral.RequireUserAgent = true
cfg.Behavioral.RequireReferer = true
```

When score exceeds threshold, IP enters penalty mode. Failed strict checks return `403` and trigger a long ban (`PenaltyBanTTL`).

## GuardGo Agent (Sidecar)

Run agent next to your API pod to expose Redis protection metrics for Prometheus/Grafana:

```bash
go run ./cmd/guardgo-agent --redis-addr 127.0.0.1:6379 --prefix guardgo --listen :9090
```

Metrics endpoint:
- `http://127.0.0.1:9090/metrics`

## Testing

```bash
go test ./...
go test -race ./tests/load -count=1
go test ./tests/load -bench BenchmarkEngineCheck -benchmem -run ^$ -benchtime=2s
```
