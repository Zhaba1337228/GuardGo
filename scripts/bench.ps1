Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "[1/3] Bloom benchmark"
go test ./pkg/bloom -run ^$ -bench BenchmarkBloomCleanRequest -benchmem -benchtime=5s -count=5

Write-Host "[2/3] DFA benchmark"
go test ./pkg/dfa -run ^$ -bench BenchmarkDFA100RulesMatch -benchmem -benchtime=5s -count=5

Write-Host "[3/3] Load benchmark (miniredis fallback)"
go test ./tests/load -run ^$ -bench BenchmarkEngineCheckParallel -benchmem -benchtime=10s -count=5

