package guardgo

import (
	"math"
	"sync"
	"time"
)

type selfHealingState struct {
	cfg SelfHealingConfig

	mu sync.Mutex

	windowStart time.Time
	total       int
	clean       int
	perIP       map[string]int

	multiplier float64
}

func newSelfHealingState(cfg SelfHealingConfig) *selfHealingState {
	return &selfHealingState{
		cfg:         cfg,
		perIP:       make(map[string]int, 128),
		multiplier:  1.0,
		windowStart: time.Now(),
	}
}

func (s *selfHealingState) effectiveLimit(ip string, baseLimit int, now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rotateWindowIfNeeded(now)

	if ip != "" {
		s.perIP[ip]++
		if s.total >= s.cfg.MinSamples {
			cleanRatio := float64(s.clean) / float64(maxInt(1, s.total))
			if cleanRatio >= s.cfg.CleanTrafficThreshold {
				ipCount := s.perIP[ip]
				trigger := float64(baseLimit) * s.cfg.SpikeFactor
				if trigger < 1 {
					trigger = 1
				}
				if float64(ipCount) > trigger {
					spikeMultiplier := float64(ipCount) / trigger
					if spikeMultiplier > s.multiplier {
						s.multiplier = math.Min(s.cfg.MaxSensitivity, spikeMultiplier)
					}
				}
			}
		}
	}

	limit := int(float64(baseLimit) / s.multiplier)
	if limit < 1 {
		return 1
	}
	return limit
}

func (s *selfHealingState) observe(decision Decision, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rotateWindowIfNeeded(now)
	s.total++
	if decision.Allowed || decision.Reason == ReasonFailOpen {
		s.clean++
	}
}

func (s *selfHealingState) rotateWindowIfNeeded(now time.Time) {
	if now.Sub(s.windowStart) < s.cfg.Window {
		return
	}
	s.windowStart = now
	s.total = 0
	s.clean = 0
	clear(s.perIP)
	if s.multiplier > 1.0 {
		s.multiplier -= s.cfg.DecayStep
		if s.multiplier < 1.0 {
			s.multiplier = 1.0
		}
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
