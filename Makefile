GO ?= go

.PHONY: test test-race bench bench-bloom bench-dfa bench-load chaos

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./tests/integration -count=1

bench: bench-bloom bench-dfa bench-load

bench-bloom:
	$(GO) test ./pkg/bloom -run ^$$ -bench BenchmarkBloomCleanRequest -benchmem -benchtime=5s -count=5

bench-dfa:
	$(GO) test ./pkg/dfa -run ^$$ -bench BenchmarkDFA100RulesMatch -benchmem -benchtime=5s -count=5

bench-load:
	$(GO) test ./tests/load -run ^$$ -bench BenchmarkEngineCheckParallel -benchmem -benchtime=10s -count=5

chaos:
	$(GO) test -tags chaos ./tests/chaos -v -count=1

