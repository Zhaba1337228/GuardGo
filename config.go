package guardgo

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

type MiddlewareConfig struct {
	// Redis is required unless you only use the local blacklist cache.
	Redis *redis.Client

	// MaxRequests is the maximum number of requests allowed per Window for a key.
	MaxRequests int
	Window      time.Duration

	// Rules are evaluated asynchronously (best-effort) to support adaptive bans.
	// Rules MUST NOT read or depend on r.Body.
	Rules []Rule

	// FailOpen allows traffic when Redis is unavailable or returns an error.
	FailOpen bool

	// Prefix is used to build Redis keys. Default: "guardgo".
	Prefix string

	// CacheSize is the local blacklist LRU size. Default: 4096.
	CacheSize int

	// DefaultBanTTL is used by adaptive banning when no rule-specific TTL exists.
	// Default: 1h.
	DefaultBanTTL time.Duration

	// AdaptiveBan enables background adaptive banning.
	AdaptiveBan AdaptiveBanConfig

	// KeyFunc defines rate-limit key. Default: IP.
	KeyFunc func(r *http.Request) string

	// IPFunc extracts the client IP for ban keys and default KeyFunc.
	// Default: RemoteAddr host part.
	IPFunc func(r *http.Request) string

	// Metrics is optional. If nil, metrics are no-ops.
	Metrics Metrics

	// Stats is optional generic telemetry collector.
	// If nil, telemetry calls are dropped.
	Stats StatsCollector

	// OnError is optional error callback (Redis/Lua/background).
	OnError func(err error)

	// DenyStatusCode is used by middleware adapters when request is blocked.
	// Default: 429.
	DenyStatusCode int

	// Bloom enables optional in-memory bloom filter short-circuit.
	// If enabled and IP is definitely not blacklisted (negative bloom test),
	// request is allowed without Redis roundtrip.
	Bloom BloomConfig

	// Tracing enables OpenTelemetry spans for request processing.
	Tracing TracingConfig

	// SelfHealing dynamically adjusts effective request limits based on behavior.
	SelfHealing SelfHealingConfig
}

type AdaptiveBanConfig struct {
	Enabled bool

	// ThreatKeyWindow is a rolling window for counting "threatful" requests per IP.
	// Default: 10m.
	ThreatKeyWindow time.Duration

	// ThreatCountThreshold triggers a ban when the per-IP threat count within
	// ThreatKeyWindow reaches this value. Default: 100.
	ThreatCountThreshold int64

	// BanTTL is the ban duration once ThreatCountThreshold is reached.
	// Default: DefaultBanTTL.
	BanTTL time.Duration

	// QueueSize is the buffered channel size for background analysis. Default: 8192.
	QueueSize int

	// Workers is the number of background workers. Default: 2 * GOMAXPROCS.
	Workers int

	// CaptureHeaders lists headers to snapshot for rules. Default: []{"User-Agent"}.
	CaptureHeaders []string
}

type BloomConfig struct {
	Enabled bool

	// Bits is bloom filter size in bits. Default: 1<<20.
	Bits uint64

	// Hashes is number of hash functions. Default: 6.
	Hashes uint8
}

type TracingConfig struct {
	Enabled bool
	Tracer  trace.Tracer
}

type SelfHealingConfig struct {
	Enabled bool

	// Window is a rolling analysis period.
	Window time.Duration

	// MinSamples controls the minimum traffic volume before adaptation.
	MinSamples int

	// CleanTrafficThreshold is the minimum clean ratio to activate spike detection.
	CleanTrafficThreshold float64

	// SpikeFactor triggers stricter limits when a single IP exceeds max*SpikeFactor.
	SpikeFactor float64

	// MaxSensitivity is upper bound of adaptive multiplier.
	MaxSensitivity float64

	// DecayStep controls how quickly sensitivity returns to normal per window.
	DecayStep float64
}

func (cfg *MiddlewareConfig) withDefaults() MiddlewareConfig {
	out := *cfg
	if out.Prefix == "" {
		out.Prefix = "guardgo"
	}
	if out.CacheSize <= 0 {
		out.CacheSize = 4096
	}
	if out.DefaultBanTTL <= 0 {
		out.DefaultBanTTL = time.Hour
	}
	if out.Window <= 0 {
		out.Window = time.Second
	}
	if out.MaxRequests <= 0 {
		out.MaxRequests = 1000
	}
	if out.IPFunc == nil {
		out.IPFunc = defaultIPFunc
	}
	if out.KeyFunc == nil {
		out.KeyFunc = func(r *http.Request) string { return out.IPFunc(r) }
	}
	if out.Metrics == nil {
		out.Metrics = NopMetrics{}
	}
	if out.Stats == nil {
		out.Stats = NopStatsCollector{}
	}
	if out.AdaptiveBan.ThreatKeyWindow <= 0 {
		out.AdaptiveBan.ThreatKeyWindow = 10 * time.Minute
	}
	if out.AdaptiveBan.ThreatCountThreshold <= 0 {
		out.AdaptiveBan.ThreatCountThreshold = 100
	}
	if out.AdaptiveBan.QueueSize <= 0 {
		out.AdaptiveBan.QueueSize = 8192
	}
	if out.AdaptiveBan.Workers <= 0 {
		out.AdaptiveBan.Workers = 0 // resolved in NewEngine to use runtime.GOMAXPROCS
	}
	if len(out.AdaptiveBan.CaptureHeaders) == 0 {
		out.AdaptiveBan.CaptureHeaders = []string{"User-Agent"}
	}
	if out.AdaptiveBan.BanTTL <= 0 {
		out.AdaptiveBan.BanTTL = out.DefaultBanTTL
	}
	if out.DenyStatusCode <= 0 {
		out.DenyStatusCode = 429
	}
	if out.Bloom.Bits == 0 {
		out.Bloom.Bits = 1 << 20
	}
	if out.Bloom.Hashes == 0 {
		out.Bloom.Hashes = 6
	}
	if out.SelfHealing.Window <= 0 {
		out.SelfHealing.Window = 30 * time.Second
	}
	if out.SelfHealing.MinSamples <= 0 {
		out.SelfHealing.MinSamples = 200
	}
	if out.SelfHealing.CleanTrafficThreshold <= 0 {
		out.SelfHealing.CleanTrafficThreshold = 0.90
	}
	if out.SelfHealing.SpikeFactor <= 0 {
		out.SelfHealing.SpikeFactor = 3.0
	}
	if out.SelfHealing.MaxSensitivity <= 1 {
		out.SelfHealing.MaxSensitivity = 4.0
	}
	if out.SelfHealing.DecayStep <= 0 {
		out.SelfHealing.DecayStep = 0.20
	}
	return out
}

// DefaultConfig returns a ready-to-use configuration with sane defaults.
func DefaultConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Redis: redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		}),
		MaxRequests: 1000,
		Window:      time.Second,
	}
}
