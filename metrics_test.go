package guardgo

import (
	"testing"
	"time"
)

func TestNopMetrics(t *testing.T) {
	t.Parallel()
	var m Metrics = NopMetrics{}
	m.IncAllowed()
	m.IncRateLimited()
	m.IncBlacklisted()
	m.IncRedisErrors()
	m.ObserveLuaLatency(time.Millisecond)
}

func TestNopStatsCollector(t *testing.T) {
	t.Parallel()
	var s StatsCollector = NopStatsCollector{}
	s.Record("any", 1.0, map[string]string{"k": "v"})
	s.Record("any", 0, nil)
}
