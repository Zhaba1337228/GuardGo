# Contributing

## Development Setup

1. Install Go `1.25+`
2. Start Redis (`127.0.0.1:6379`) for integration checks
3. Run:

```bash
go test ./...
```

## Test Matrix

- Unit + integration:
```bash
go test ./...
```

- Race:
```bash
go test -race ./tests/integration -count=1
```

- Benchmarks:
```bash
go test ./pkg/bloom -run ^$ -bench . -benchmem
go test ./pkg/dfa -run ^$ -bench . -benchmem
go test ./tests/load -run ^$ -bench . -benchmem
```

- Chaos (requires Docker):
```bash
go test -tags chaos ./tests/chaos -v -count=1
```

## Pull Request Guidelines

- Keep API changes backward-compatible unless explicitly discussed.
- Add/adjust tests for each behavior change.
- Keep comments concise and focused on non-obvious logic.
- Update `README.md` and `docs/` if behavior or structure changes.

