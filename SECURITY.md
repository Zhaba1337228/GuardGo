# Security Policy

## Reporting a Vulnerability

If you discover a security issue, please report it privately to the maintainers.
Do not open a public issue with exploit details before maintainers confirm a fix plan.

Recommended report content:
- affected version/commit
- reproduction steps
- expected impact
- optional mitigation suggestions

## Scope

Security-sensitive areas:
- request decision pipeline (`engine.go`)
- Redis Lua scripts (`internal/redislua/`)
- penalty and blacklist transitions
- sidecar/CLI operational surfaces

