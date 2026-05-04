<div align="center">

# GuardGo

**Go 高吞吐 API 防护库,Redis 驱动。**

原子 Lua 关键路径。信誉评分流水线。行为熵分析。热加载签名。零分配快路径。

[![CI](https://github.com/Zhaba1337228/GuardGo/actions/workflows/ci.yml/badge.svg)](https://github.com/Zhaba1337228/GuardGo/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Zhaba1337228/GuardGo/branch/main/graph/badge.svg)](https://codecov.io/gh/Zhaba1337228/GuardGo)
[![Go Reference](https://pkg.go.dev/badge/github.com/Zhaba1337228/GuardGo.svg)](https://pkg.go.dev/github.com/Zhaba1337228/GuardGo)
[![License](https://img.shields.io/github/license/Zhaba1337228/GuardGo.svg)](../LICENSE)

[English](../README.md) · [Русский](README.ru.md) · **中文**

</div>

---

## ✨ 特性

| | |
|---|---|
| ⚡ **干净请求 ~12 ns** | Bloom 过滤器零分配快路径 |
| 🛡️ **原子 Redis Lua** | 黑名单 + 限速一次往返完成 |
| 🧠 **信誉评分流水线** | 静态规则 + Evaluator 评分 + DFA 签名 + 行为熵 |
| 🔥 **无重启热加载** | `SIGHUP` 触发签名热替换,连接不中断 |
| 🪜 **动态退避封禁** | TTL 阶梯:`1m` → `10m` → `24h` |
| 🧯 **自愈式限速** | 高负载自动收紧,清流时自动恢复 |
| 📊 **开箱即用可观测性** | OpenTelemetry span、Prometheus sidecar、通用 stats hook |
| 🧩 **即插即用中间件** | `net/http`、`gin`、`echo`、`fiber` 各一行 |

---

## 📦 安装

```bash
go get github.com/Zhaba1337228/GuardGo
```

> **需要 Go 1.25+** · **Redis 6.2+ / Valkey / KeyDB / Dragonfly**

---

## 🚀 快速上手

### 零配置

```go
guard := guardgo.New(guardgo.DefaultConfig())
defer guard.Close()
```

> 默认:Redis `127.0.0.1:6379`,限速 `1000 req/s`。

### `net/http`

```go
rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
cfg := guardgo.NewConfig(rdb, 200, time.Second)
cfg.FailOpen = true
cfg.Bloom.Enabled = true
cfg.Reputation.Enabled = true

engine := guardgo.New(cfg)
defer engine.Close()

http.ListenAndServe(":8080", engine.Middleware(mux))
```

### 框架适配器

```go
router.Use(guardgo.Gin(engine))
e.Use(guardgo.Echo(engine))
app.Use(guardgo.Fiber(engine))
```

---

## 🛡️ 防护模型

GuardGo 为每个请求计算 **score** 并决策:

| Score | 动作 |
|---|---|
| `< WarningLevel` | 正常通过 |
| `≥ WarningLevel` | 进入惩罚模式(强制 UA/Referer 校验,收紧限速) |
| `≥ Threshold` | 通过 Redis Lua 加入黑名单,使用阶梯式 TTL |

**指纹:** `IP + User-Agent + Accept-Language`(小写归一化后哈希)。

---

## 🔥 热加载

配置 `RulesetFile` 或 `DynamicRules` 后,签名可通过 POSIX `SIGHUP` 热加载:

```bash
kill -HUP <pid>
```

无需重启进程,不丢失连接。

---

## 📊 可观测性

- OpenTelemetry spans (`guardgo.process`)
- Prometheus sidecar:`cmd/guardgo-agent`
- 终端实时面板:`cmd/guardgo-cli`
- 通用 `StatsCollector` 接口

---

## ⚡ 基准测试

> 12th Gen Intel i5-12400 · windows/amd64 · 2026-03-08

| 场景 | 延迟 | 吞吐 | 分配 |
|---|---:|---:|---:|
| 干净请求 (Bloom) | **11.8 ns** | ~84 M ops/s | `0 B/op` `0 allocs/op` |
| DFA 匹配 (100 条规则) | **454 ns** | ~2.2 M ops/s | `24 B/op` `1 alloc/op` |
| Redis 兜底 (并行) | 205–225 µs | ~4.4 K ops/s/核 | ~206 KB/op |

---

## 📚 文档

- [架构](ARCHITECTURE.md)
- [项目结构](PROJECT_LAYOUT.md)
- [API](API.md)
- [示例](../examples/)

---

## 🤝 贡献

欢迎 PR。请阅读 [CONTRIBUTING.md](../CONTRIBUTING.md) 和 [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md)。
强制使用 Conventional Commits — release-please 据此驱动 `CHANGELOG.md`。

## 🔒 安全

发现漏洞请阅读 [SECURITY.md](../SECURITY.md)。请不要在公开 issue 中泄露利用细节。

## 📄 协议

[MIT](../LICENSE) © GuardGo contributors
