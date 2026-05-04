<div align="center">

# GuardGo

**Высокопроизводительная защита API на Go с Redis под капотом.**

Атомарный Lua-критпуть. Репутационный пайплайн. Поведенческая энтропия. Горячая перезагрузка сигнатур. Zero-allocation быстрый путь.

[![CI](https://github.com/Zhaba1337228/GuardGo/actions/workflows/ci.yml/badge.svg)](https://github.com/Zhaba1337228/GuardGo/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Zhaba1337228/GuardGo/branch/main/graph/badge.svg)](https://codecov.io/gh/Zhaba1337228/GuardGo)
[![Go Reference](https://pkg.go.dev/badge/github.com/Zhaba1337228/GuardGo.svg)](https://pkg.go.dev/github.com/Zhaba1337228/GuardGo)
[![License](https://img.shields.io/github/license/Zhaba1337228/GuardGo.svg)](../LICENSE)

[English](../README.md) · **Русский** · [中文](README.zh.md)

</div>

---

## ✨ Главное

| | |
|---|---|
| ⚡ **~12 нс на чистый запрос** | Bloom-фильтр срезает горячий путь без аллокаций |
| 🛡️ **Атомарный Redis Lua** | Чёрный список + rate-limit одним round-trip |
| 🧠 **Репутационный пайплайн** | Правила + evaluator-скоринг + DFA-сигнатуры + энтропия поведения |
| 🔥 **Hot-reload без рестарта** | Перезагрузка правил по `SIGHUP`, соединения не рвутся |
| 🪜 **Динамический backoff** | TTL банов растут: `1m` → `10m` → `24h` |
| 🧯 **Self-healing лимиты** | Сам сужает лимит под нагрузкой, расслабляется при чистом трафике |
| 📊 **Готовая observability** | OpenTelemetry spans, Prometheus sidecar, generic stats-hook |
| 🧩 **Drop-in middleware** | `net/http`, `gin`, `echo`, `fiber` — одна строка |

---

## 📦 Установка

```bash
go get github.com/Zhaba1337228/GuardGo
```

> **Go 1.25+** · **Redis 6.2+ / Valkey / KeyDB / Dragonfly**

---

## 🚀 Быстрый старт

### Zero-config

```go
guard := guardgo.New(guardgo.DefaultConfig())
defer guard.Close()
```

> Дефолты: Redis `127.0.0.1:6379`, лимит `1000 req/s`.

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

### Адаптеры под фреймворки

```go
router.Use(guardgo.Gin(engine))
e.Use(guardgo.Echo(engine))
app.Use(guardgo.Fiber(engine))
```

---

## 🛡️ Модель защиты

GuardGo считает каждому запросу **score** и решает:

| Score | Действие |
|---|---|
| `< WarningLevel` | Обычный режим |
| `≥ WarningLevel` | Penalty-mode (строгие проверки UA/Referer, ужесточённый лимит) |
| `≥ Threshold` | Бан через Redis Lua с эскалирующим TTL |

**Fingerprint:** `IP + User-Agent + Accept-Language` (lower-case, хешируется).

---

## 🔥 Hot Reload

Если в конфиге задан `RulesetFile` или `DynamicRules`, сигнатуры перезагружаются по POSIX `SIGHUP`:

```bash
kill -HUP <pid>
```

Без рестарта. Без потери соединений.

---

## 📊 Observability

- OpenTelemetry spans (`guardgo.process`)
- Prometheus sidecar: `cmd/guardgo-agent`
- Терминальный дашборд: `cmd/guardgo-cli`
- Generic `StatsCollector` для своего стека

---

## ⚡ Бенчмарки

> 12th Gen Intel i5-12400 · windows/amd64 · 2026-03-08

| Кейс | Latency | Throughput | Аллокации |
|---|---:|---:|---:|
| Чистый запрос (Bloom) | **11.8 нс** | ~84 М ops/s | `0 B/op` `0 allocs/op` |
| DFA-матч (100 правил) | **454 нс** | ~2.2 М ops/s | `24 B/op` `1 alloc/op` |
| Redis fallback (параллельно) | 205–225 мкс | ~4.4 K ops/s/ядро | ~206 KB/op |

---

## 📚 Документация

- [Архитектура](ARCHITECTURE.md)
- [Структура проекта](PROJECT_LAYOUT.md)
- [API](API.md)
- [Примеры](../examples/)

---

## 🤝 Контрибьют

PR'ы приветствуются. См. [CONTRIBUTING.md](../CONTRIBUTING.md) и [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md).
Conventional commits обязательны — release-please использует их для `CHANGELOG.md`.

## 🔒 Безопасность

Нашёл уязвимость — см. [SECURITY.md](../SECURITY.md). Не открывай публичные issues с эксплойт-деталями.

## 📄 Лицензия

[MIT](../LICENSE) © GuardGo contributors
