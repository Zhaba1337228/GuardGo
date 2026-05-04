# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions CI workflow (`build`, `vet`, `test`, `test -race`, coverage, lint, smoke benchmarks).
- CodeQL workflow (Go + Actions, weekly schedule).
- Dependabot config for Go modules and GitHub Actions with grouped updates.
- `.editorconfig` for consistent indentation across editors.
- `.golangci.yml` (golangci-lint v2 schema, Go 1.25 target).
- `CHANGELOG.md`, `CODE_OF_CONDUCT.md`.
- Issue templates (`bug`, `feature`) and PR template.
- `LICENSE` (MIT).

### Changed
- Removed redundant root `ARCHITECTURE.md`; canonical doc lives at `docs/ARCHITECTURE.md`.

### Fixed
- `tests/load/redis_fallback_benchmark_test.go`: explicitly ignore `rdb.Close()` error to satisfy `errcheck`.
