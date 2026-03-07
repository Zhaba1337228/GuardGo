package guardgo

import "time"

type Metrics interface {
	IncAllowed()
	IncRateLimited()
	IncBlacklisted()
	IncRedisErrors()
	ObserveLuaLatency(d time.Duration)
}

type NopMetrics struct{}

func (NopMetrics) IncAllowed()                       {}
func (NopMetrics) IncRateLimited()                   {}
func (NopMetrics) IncBlacklisted()                   {}
func (NopMetrics) IncRedisErrors()                   {}
func (NopMetrics) ObserveLuaLatency(d time.Duration) {}
