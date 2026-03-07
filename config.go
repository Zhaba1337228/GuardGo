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

	// Evaluators calculate additive reputation score.
	Evaluators []Evaluator

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

	// RulesetFile points to JSON/YAML signatures loaded at startup.
	RulesetFile string

	// DynamicRules allows hot-swapping signatures without recreating Engine.
	DynamicRules *RulesetManager

	// AutoReloadOnSIGHUP enables ruleset hot reload on POSIX SIGHUP signal.
	// Applicable when RulesetFile or DynamicRules is configured.
	AutoReloadOnSIGHUP bool

	// Behavioral enables risk scoring and penalty-box flow.
	Behavioral BehavioralConfig

	// Reputation controls score pipeline and dynamic backoff bans.
	Reputation ReputationConfig
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

type BehavioralConfig struct {
	Enabled bool

	// RiskWindow is TTL for per-IP risk score.
	RiskWindow time.Duration

	// ScoreThreshold pushes IP into penalty box.
	ScoreThreshold int64

	// PenaltyTTL is duration of penalty box mode.
	PenaltyTTL time.Duration

	// PenaltyBanTTL is ban duration after failed penalty checks.
	PenaltyBanTTL time.Duration

	// RequireUserAgent enforces User-Agent header in penalty mode.
	RequireUserAgent bool

	// RequireReferer enforces Referer header in penalty mode.
	RequireReferer bool

	// QueryDiversityThreshold adds risk when query diversity spikes.
	QueryDiversityThreshold int

	// HeaderDiversityThreshold adds risk when header diversity spikes.
	HeaderDiversityThreshold int

	// DiversityWeight contributes to risk when thresholds exceeded.
	DiversityWeight int64
}

type ReputationConfig struct {
	Enabled bool

	// WarningLevel moves actor into penalty box.
	WarningLevel float64

	// Threshold triggers blacklist via reputation Lua script.
	Threshold float64

	// ScoreWindow controls reputation hash TTL.
	ScoreWindow time.Duration

	// PenaltyTTL is strict-mode duration.
	PenaltyTTL time.Duration

	// BackoffWindow controls how long attack attempt counter lives.
	BackoffWindow time.Duration

	// BanLevel1 is ban duration for first serious attack.
	BanLevel1 time.Duration

	// BanLevel2 is ban duration for second serious attack.
	BanLevel2 time.Duration

	// BanLevel3 is ban duration for third and next attack.
	BanLevel3 time.Duration
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
	if out.Behavioral.RiskWindow <= 0 {
		out.Behavioral.RiskWindow = 15 * time.Minute
	}
	if out.Behavioral.ScoreThreshold <= 0 {
		out.Behavioral.ScoreThreshold = 100
	}
	if out.Behavioral.PenaltyTTL <= 0 {
		out.Behavioral.PenaltyTTL = 30 * time.Minute
	}
	if out.Behavioral.PenaltyBanTTL <= 0 {
		out.Behavioral.PenaltyBanTTL = 24 * time.Hour
	}
	if out.Behavioral.QueryDiversityThreshold <= 0 {
		out.Behavioral.QueryDiversityThreshold = 30
	}
	if out.Behavioral.HeaderDiversityThreshold <= 0 {
		out.Behavioral.HeaderDiversityThreshold = 20
	}
	if out.Behavioral.DiversityWeight <= 0 {
		out.Behavioral.DiversityWeight = 5
	}
	if (out.RulesetFile != "" || out.DynamicRules != nil) && !out.AutoReloadOnSIGHUP {
		out.AutoReloadOnSIGHUP = true
	}
	if out.Reputation.WarningLevel <= 0 {
		out.Reputation.WarningLevel = 60
	}
	if out.Reputation.Threshold <= 0 {
		out.Reputation.Threshold = 100
	}
	if out.Reputation.ScoreWindow <= 0 {
		out.Reputation.ScoreWindow = 15 * time.Minute
	}
	if out.Reputation.PenaltyTTL <= 0 {
		out.Reputation.PenaltyTTL = 30 * time.Minute
	}
	if out.Reputation.BackoffWindow <= 0 {
		out.Reputation.BackoffWindow = 24 * time.Hour
	}
	if out.Reputation.BanLevel1 <= 0 {
		out.Reputation.BanLevel1 = time.Minute
	}
	if out.Reputation.BanLevel2 <= 0 {
		out.Reputation.BanLevel2 = 10 * time.Minute
	}
	if out.Reputation.BanLevel3 <= 0 {
		out.Reputation.BanLevel3 = 24 * time.Hour
	}
	if out.Reputation.Enabled {
		out.Behavioral.Enabled = true
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
