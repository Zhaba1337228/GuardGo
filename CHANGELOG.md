# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0](https://github.com/Zhaba1337228/GuardGo/compare/v1.0.0...v1.1.0) (2026-05-04)


### Features

* add fuzz tests and mutation testing documentation; update dependencies ([bbe0fb1](https://github.com/Zhaba1337228/GuardGo/commit/bbe0fb1c8dd7743a7d776df41c4d6753d72e1422))


### Bug Fixes

* improve determinism check in hash64 test with detailed failure message ([4a828ca](https://github.com/Zhaba1337228/GuardGo/commit/4a828ca620b4dcb4bc8adbb959f5b3c143bd231f))

## 1.0.0 (2026-05-04)


### Features

* Add initial implementation of GuardGo engine with Redis integration ([130c362](https://github.com/Zhaba1337228/GuardGo/commit/130c3626453ff98126ba532605c23c3ddc20dcc4))
* enhance CI workflow with integration tests and improved coverage reporting ([d215e5f](https://github.com/Zhaba1337228/GuardGo/commit/d215e5f57381c15b4e884a499faf3d6e41188890))
* fix version golang ([b8ad020](https://github.com/Zhaba1337228/GuardGo/commit/b8ad0202070565376aaf3d75b9b98c48a2a87cba))
* Integrate Codecov for coverage reporting and update import paths to reflect new module name ([9276628](https://github.com/Zhaba1337228/GuardGo/commit/9276628eb5ed6dab527974554a7b55d9fbf96bf3))


### Bug Fixes

* reorder import statements for consistency across multiple files ([5aba5ee](https://github.com/Zhaba1337228/GuardGo/commit/5aba5eedae0238ae9265690410bea03381dba1bb))

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
