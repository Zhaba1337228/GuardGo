# Mutation Testing

We use [gremlins](https://github.com/go-gremlins/gremlins) to verify that the
test suite actually catches behavioral changes — not just compiles and exits 0.

## Install

```bash
go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
```

## Run on a single package

```bash
# Fast smoke run on hot-path code
gremlins unleash ./pkg/dfa --tags=""

# With coverage threshold (fail if mutation score < 70%)
gremlins unleash ./pkg/dfa --threshold-efficacy=70
```

## Run on the root package

The root package includes pure helpers (`hash64`, `fingerprint`, `parseLuaResult`,
`Decision` constructors) — ideal mutation targets:

```bash
gremlins unleash . --tags="" --output=mutation-report.json
```

## What gremlins changes

It rewrites operators and constants in your code (e.g. `>` → `>=`, `+` → `-`,
`true` → `false`) and re-runs the tests. If the tests still pass, the mutation
"survived" — meaning either the line is dead code or your tests don't actually
assert on its behavior. Aim for **mutation score ≥ 70%** on hot-path packages.

## CI integration

Mutation testing is slow (each mutant re-runs the full test suite), so it is
**not** part of the default CI. Run it locally before major refactors, or add
a manual workflow:

```yaml
name: mutation
on:
  workflow_dispatch:
jobs:
  gremlins:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.x"
      - run: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
      - run: gremlins unleash ./pkg/... .
```
