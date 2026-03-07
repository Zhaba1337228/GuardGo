package guardgo

// StatsCollector is a vendor-neutral contract for custom telemetry export.
// Implement this interface to forward metrics to Prometheus, Datadog, OTEL, etc.
type StatsCollector interface {
	Record(name string, value float64, labels map[string]string)
}

type NopStatsCollector struct{}

func (NopStatsCollector) Record(name string, value float64, labels map[string]string) {}
