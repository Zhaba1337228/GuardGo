package guardgo

import (
	"hash/fnv"
	"net/http"
	"sync"
	"time"
)

type behaviorTracker struct {
	cfg BehavioralConfig

	mu    sync.Mutex
	store map[string]*ipBehavior
}

type ipBehavior struct {
	expiresAt time.Time
	queries   map[uint64]struct{}
	headers   map[uint64]struct{}
}

func newBehaviorTracker(cfg BehavioralConfig) *behaviorTracker {
	return &behaviorTracker{
		cfg:   cfg,
		store: make(map[string]*ipBehavior, 256),
	}
}

func (t *behaviorTracker) observe(ip string, r *http.Request, now time.Time) int64 {
	if ip == "" || r == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.store[ip]
	if !ok || now.After(entry.expiresAt) {
		entry = &ipBehavior{
			expiresAt: now.Add(t.cfg.RiskWindow),
			queries:   make(map[uint64]struct{}, 32),
			headers:   make(map[uint64]struct{}, 32),
		}
		t.store[ip] = entry
	}

	if r.URL != nil && r.URL.RawQuery != "" {
		entry.queries[hash64(r.URL.RawQuery)] = struct{}{}
	}
	headerSig := headerSignature(r.Header)
	if headerSig != 0 {
		entry.headers[headerSig] = struct{}{}
	}

	var score int64
	if len(entry.queries) >= t.cfg.QueryDiversityThreshold {
		score += t.cfg.DiversityWeight
	}
	if len(entry.headers) >= t.cfg.HeaderDiversityThreshold {
		score += t.cfg.DiversityWeight
	}
	return score
}

func headerSignature(h http.Header) uint64 {
	if len(h) == 0 {
		return 0
	}
	hasher := fnv.New64a()
	for key, values := range h {
		_, _ = hasher.Write([]byte(key))
		for _, value := range values {
			_, _ = hasher.Write([]byte(value))
		}
	}
	return hasher.Sum64()
}

func hash64(s string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(s))
	return hasher.Sum64()
}
