<div align="center">

# GuardGo

**High-throughput, Redis-backed API protection for Go.**

Atomic Lua critical path. Reputation pipeline. Behavioral entropy. Hot-reloadable signatures. Zero-allocation fast path.

[![CI](https://github.com/Zhaba1337228/GuardGo/actions/workflows/ci.yml/badge.svg)](https://github.com/Zhaba1337228/GuardGo/actions/workflows/ci.yml)
[![CodeQL](https://github.com/Zhaba1337228/GuardGo/actions/workflows/codeql.yml/badge.svg)](https://github.com/Zhaba1337228/GuardGo/actions/workflows/codeql.yml)
[![govulncheck](https://github.com/Zhaba1337228/GuardGo/actions/workflows/govulncheck.yml/badge.svg)](https://github.com/Zhaba1337228/GuardGo/actions/workflows/govulncheck.yml)
[![codecov](https://codecov.io/gh/Zhaba1337228/GuardGo/branch/main/graph/badge.svg)](https://codecov.io/gh/Zhaba1337228/GuardGo)
[![Go Reference](https://pkg.go.dev/badge/github.com/Zhaba1337228/GuardGo.svg)](https://pkg.go.dev/github.com/Zhaba1337228/GuardGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/Zhaba1337228/GuardGo)](https://goreportcard.com/report/github.com/Zhaba1337228/GuardGo)
[![License](https://img.shields.io/github/license/Zhaba1337228/GuardGo.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Zhaba1337228/GuardGo.svg)](go.mod)
[![Release](https://img.shields.io/github/v/release/Zhaba1337228/GuardGo?include_prereleases&sort=semver)](https://github.com/Zhaba1337228/GuardGo/releases)

**English** · [Русский](docs/README.ru.md) · [中文](docs/README.zh.md)

</div>

---

## ✨ Highlights

| | |
|---|---|
| ⚡ **~12 ns clean-request fast path** | Bloom filter short-circuits hot paths with zero allocations |
| 🛡️ **Atomic Redis Lua critical path** | Blacklist + rate-limit decision in a single round-trip |
| 🧠 **Reputation pipeline** | Static rules + score evaluators + DFA signatures + behavioral entropy |
| 🔥 **Hot-reload without restart** | `SIGHUP`-driven ruleset swap, no dropped connections |
| 🪜 **Dynamic backoff** | Escalating ban TTLs: `1m` → `10m` → `24h` |
| 🧯 **Self-healing limits** | Auto-tightens under load, decays back when traffic clears |
| 📊 **Observability built-in** | OpenTelemetry spans, Prometheus sidecar, generic stats hook |
| 🧩 **Drop-in middleware** | `net/http`, `gin`, `echo`, `fiber` — one line each |

---

## 📦 Install

```bash
go get github.com/Zhaba1337228/GuardGo
```

> **Requires Go 1.25+** · **Redis 6.2+ / Valkey / KeyDB / Dragonfly**

---

## 🚀 Quick Start

### Zero-config

```go
guard := guardgo.New(guardgo.DefaultConfig())
defer guard.Close()
```

> Defaults: Redis at `127.0.0.1:6379`, `1000 req/s`.

### `net/http`

```go
package main

import (
    "net/http"
    "time"

    guardgo "github.com/Zhaba1337228/GuardGo"
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

### Framework adapters

```go
router.Use(guardgo.Gin(engine))
e.Use(guardgo.Echo(engine))
app.Use(guardgo.Fiber(engine))
```

---

## 🔬 How It Works

```
                ┌──────────────────────────────┐
  request ───▶  │  Local LRU blacklist cache   │  ─── hit ──▶ block (cached)
                └──────────────────────────────┘
                           │ miss
                           ▼
                ┌──────────────────────────────┐
                │  Penalty box check           │  ─── strict mode (UA/Referer required)
                └──────────────────────────────┘
                           │ pass
                           ▼
                ┌──────────────────────────────┐
                │  Bloom filter (negative)     │  ─── definitely-not-banned ──▶ allow
                └──────────────────────────────┘
                           │ unknown
                           ▼
                ┌──────────────────────────────┐
                │  Redis Lua (atomic):         │
                │   • blacklist set check      │
                │   • rate-limit counter       │
                └──────────────────────────────┘
                           │
                           ▼
                       allow / block
                           │
                           ▼ (async, off critical path)
                ┌──────────────────────────────┐
                │  Reputation pipeline:        │
                │   • static Rule checks       │
                │   • Evaluator score          │
                │   • DFA signatures           │
                │   • behavioral entropy       │
                │   ─▶ penalty / blacklist     │
                └──────────────────────────────┘
```

---

## 🛡️ Security Model

GuardGo evaluates traffic through a **score pipeline**:

| Score | Action |
|---|---|
| `< WarningLevel` | Normal flow |
| `≥ WarningLevel` | Penalty mode (strict UA/Referer, tightened limit) |
| `≥ Threshold` | Blacklist via Redis Lua, escalating ban TTL |

**Fingerprint:** `IP + User-Agent + Accept-Language` (case-folded, hashed).

**Dynamic backoff** (per IP, sliding window):

| Attempt | Default TTL |
|---|---|
| #1 | `1m` |
| #2 | `10m` |
| #3+ | `24h` |

---

## 🔥 Hot Reload

If `RulesetFile` or `DynamicRules` is configured, signatures hot-reload on POSIX `SIGHUP`:

```bash
kill -HUP <pid>
```

No process restart. No dropped connections.

### Built-in presets

```go
rules.ApplyDefaultSecurityPresets(&cfg)
```

Bundled signatures: SQLi probes, path traversal, scanner UA fingerprints, common exploit paths.

---

## 📊 Observability

| Surface | What you get |
|---|---|
| **OpenTelemetry** | `guardgo.process` span with decision attributes |
| **Prometheus sidecar** | `cmd/guardgo-agent` exposes `/metrics` |
| **Live dashboard** | `cmd/guardgo-cli` — terminal UI: top risk, active blocks, DFA stats |
| **Generic hook** | `StatsCollector` interface — forward to Datadog / OTEL / custom |

```bash
go run ./cmd/guardgo-agent --redis-addr 127.0.0.1:6379 --prefix guardgo --listen :9090
go run ./cmd/guardgo-cli   --redis-addr 127.0.0.1:6379 --prefix guardgo --ruleset ./rules.yaml
```

---

## ⚡ Benchmarks

> 12th Gen Intel(R) Core(TM) i5-12400 · windows/amd64 · 2026-03-08

| Case | Latency | Throughput | Allocations |
|---|---:|---:|---:|
| Clean Request (Bloom) | **11.8 ns** | ~84 M ops/s | `0 B/op` `0 allocs/op` |
| DFA Match (100 rules) | **454 ns** | ~2.2 M ops/s | `24 B/op` `1 alloc/op` |
| Redis Fallback (parallel, miniredis) | 205–225 µs | ~4.4 K ops/s/core | ~206 KB/op |

Reproduce:

```bash
go test ./pkg/bloom -run ^$ -bench BenchmarkBloomCleanRequest -benchmem -benchtime=5s -count=5
go test ./pkg/dfa   -run ^$ -bench BenchmarkDFA100RulesMatch  -benchmem -benchtime=5s -count=5
go test ./tests/load -run ^$ -bench BenchmarkEngineCheckParallel -benchmem -benchtime=10s -count=5

# Real Redis fallback (recommended for production profiling)
REDIS_ADDR=127.0.0.1:6379 \
  go test ./tests/load -run ^$ -bench BenchmarkRedisFallbackRealRedis -benchmem -benchtime=10s -count=5
```

---

## 🧪 Testing

```bash
go test ./...                                          # all unit + integration (miniredis)
go test -race ./tests/integration -count=1             # data races
REDIS_ADDR=127.0.0.1:6379 go test ./tests/load -bench .  # real Redis perf
go test -tags chaos ./tests/chaos -v -count=1          # chaos (requires Docker)
```

---

## 🗂️ Project Layout

```
.
├── engine*.go           # core decision pipeline (split by concern)
├── decision.go          # Reason / Decision types and headers
├── config.go            # MiddlewareConfig + defaults
├── frameworks.go        # Gin / Echo / Fiber adapters
├── middleware_http.go   # net/http middleware
├── ruleset.go           # YAML/JSON ruleset loader + hot-reload
├── pkg/
│   ├── bloom/           # zero-alloc bloom filter
│   └── dfa/             # multi-pattern DFA matcher
├── internal/
│   ├── blacklist/       # local LRU
│   └── redislua/        # embedded Lua scripts
├── rules/               # built-in rules and presets
├── cmd/
│   ├── guardgo-agent/   # Prometheus sidecar
│   └── guardgo-cli/     # live dashboard
├── examples/            # net/http, gin, presets
└── tests/
    ├── integration/     # miniredis-driven
    ├── load/            # benchmarks
    └── chaos/           # testcontainers-driven (build tag: chaos)
```

---

## 📚 Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Project layout](docs/PROJECT_LAYOUT.md)
- [API methods](docs/API.md)
- [Examples](examples/)

---

## 🤝 Contributing

PRs welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

Conventional commits are required (`feat:`, `fix:`, `perf:`, `docs:`, `test:`, `ci:`, `chore:`) — release-please uses them to drive `CHANGELOG.md` and version bumps.

## 🔒 Security

Found a vulnerability? See [SECURITY.md](SECURITY.md). Please **do not** open public issues with exploit details before a fix is in place.

## 📄 License

[MIT](LICENSE) © GuardGo contributors
