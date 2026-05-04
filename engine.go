package guardgo

import (
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"guardgo/internal/blacklist"
	"guardgo/internal/redislua"
	"guardgo/pkg/bloom"
)

// Engine is the request-decision core. It is safe for concurrent use after
// NewEngine returns and until Close is called.
type Engine struct {
	cfg MiddlewareConfig

	cache *blacklist.Cache
	pbox  *blacklist.Cache
	bf    *bloom.Filter
	sh    *selfHealingState

	ruleset  *RulesetManager
	compiled *CompiledRuleset
	behavior *behaviorTracker

	bgStop chan struct{}
	bgWg   sync.WaitGroup

	reqPool sync.Pool

	analysisCh chan *requestSnapshot

	hotReloadStop func()
}

// requestSnapshot is a pooled, header-stable copy of the parts of an
// *http.Request that the async pipeline needs after the request returns.
type requestSnapshot struct {
	ip     string
	key    string
	method string
	host   string
	path   string
	rawQ   string
	ua     string
	lang   string
	hdr    http.Header
	ts     time.Time
}

// Reference imports kept here so split files don't drift if a sibling file
// stops using a particular package.
var (
	_ = redislua.CriticalPathScript
	_ = bloom.New
)

// NewEngine constructs an Engine, applying defaults and starting background
// workers if any reactive subsystem (rules, reputation, behavior, ruleset
// reload) is configured.
func NewEngine(cfg MiddlewareConfig) (*Engine, error) {
	cfg = cfg.withDefaults()
	if cfg.Redis == nil {
		cfg.Redis = DefaultConfig().Redis
	}
	c, err := blacklist.New(cfg.CacheSize)
	if err != nil {
		return nil, err
	}
	pboxCache, err := blacklist.New(cfg.CacheSize)
	if err != nil {
		return nil, err
	}

	e := &Engine{
		cfg:    cfg,
		cache:  c,
		pbox:   pboxCache,
		bgStop: make(chan struct{}),
		reqPool: sync.Pool{
			New: func() any { return new(requestSnapshot) },
		},
	}
	if cfg.Bloom.Enabled {
		e.bf = bloom.New(cfg.Bloom.Bits, cfg.Bloom.Hashes)
	}
	if cfg.SelfHealing.Enabled {
		e.sh = newSelfHealingState(cfg.SelfHealing)
	}
	if cfg.Behavioral.Enabled {
		e.behavior = newBehaviorTracker(cfg.Behavioral)
	}
	if cfg.DynamicRules != nil {
		e.ruleset = cfg.DynamicRules
		e.compiled = cfg.DynamicRules.Current()
	}
	if cfg.CompiledRules != nil {
		e.compiled = cfg.CompiledRules
	}
	if cfg.RulesetFile != "" && e.ruleset == nil {
		manager, loadErr := NewRulesetManager(cfg.RulesetFile)
		if loadErr != nil {
			return nil, loadErr
		}
		e.ruleset = manager
		e.compiled = manager.Current()
	}

	shouldRunBackground := (cfg.AdaptiveBan.Enabled && len(cfg.Rules) > 0) ||
		cfg.Behavioral.Enabled ||
		cfg.Reputation.Enabled ||
		len(cfg.Evaluators) > 0 ||
		cfg.RulesetFile != "" ||
		cfg.DynamicRules != nil
	if shouldRunBackground {
		workers := cfg.AdaptiveBan.Workers
		if workers <= 0 {
			workers = 2 * runtime.GOMAXPROCS(0)
		}
		e.analysisCh = make(chan *requestSnapshot, cfg.AdaptiveBan.QueueSize)
		e.startBackground(workers)
	}
	e.startSignalHotReload()

	return e, nil
}

// Close stops background workers and releases attached resources.
// Safe to call multiple times.
func (e *Engine) Close() {
	if e.hotReloadStop != nil {
		e.hotReloadStop()
	}
	if e.analysisCh == nil {
		return
	}
	close(e.bgStop)
	e.bgWg.Wait()
}

// Config returns effective engine configuration with defaults applied.
func (e *Engine) Config() MiddlewareConfig {
	return e.cfg
}

func (e *Engine) shouldEnqueueAnalysis() bool {
	return e.analysisCh != nil
}

func (e *Engine) recordRequestStat(status, source string) {
	e.recordStat("requests_total", 1, map[string]string{
		"status": status,
		"source": source,
	})
}

func (e *Engine) recordStat(name string, value float64, labels map[string]string) {
	e.cfg.Stats.Record(name, value, labels)
}

func defaultIPFunc(r *http.Request) string {
	ra := r.RemoteAddr
	if ra == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(ra)
	if err == nil {
		return host
	}
	return ra
}
