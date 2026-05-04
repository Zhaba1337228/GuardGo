package rules

import (
	"testing"

	guardgo "github.com/Zhaba1337228/GuardGo"
)

func TestDefaultSecurityPresets(t *testing.T) {
	presets := DefaultSecurityPresets()
	if len(presets) < 5 {
		t.Fatalf("expected at least 5 preset rules, got %d", len(presets))
	}
	for _, preset := range presets {
		if preset.Name == "" || preset.Pattern == "" || preset.Weight <= 0 {
			t.Fatalf("invalid preset detected: %+v", preset)
		}
	}
}

func TestApplyDefaultSecurityPresets(t *testing.T) {
	cfg := &guardgo.MiddlewareConfig{}
	ApplyDefaultSecurityPresets(cfg)
	if cfg.CompiledRules == nil {
		t.Fatalf("expected compiled rules to be set")
	}
	stats := cfg.CompiledRules.Stats()
	if stats.TotalRules == 0 {
		t.Fatalf("expected non-empty compiled rules stats")
	}
}
