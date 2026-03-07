package rules

import "testing"

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
