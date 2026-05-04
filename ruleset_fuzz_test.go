package guardgo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoadRulesetFile_JSON makes sure arbitrary JSON content for the ruleset
// loader never panics and either yields an error or a non-nil compiled set.
func FuzzLoadRulesetFile_JSON(f *testing.F) {
	seeds := []string{
		`{"rules":[{"name":"a","match":"any","pattern":"select","weight":1}]}`,
		`{"rules":[{"name":"","match":"path","pattern":"(?i)(union|select)","weight":-1}]}`,
		`{}`,
		`{"rules":null}`,
		`{"rules":[{"match":"???","pattern":""}]}`,
		`not json`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		var spec rulesetFile
		if err := json.Unmarshal(body, &spec); err != nil {
			return // invalid JSON is acceptable
		}
		// Compile must never panic.
		c := CompileRuleset(spec.Rules)
		if c == nil {
			t.Fatal("CompileRuleset returned nil for valid JSON")
		}
		// Stats must be safe and self-consistent.
		stats := c.Stats()
		if stats.TotalRules != stats.AnyRules+stats.QueryRules+stats.HeaderRules+stats.PathRules {
			t.Fatalf("totals diverged: %+v", stats)
		}
	})
}

// FuzzLoadRulesetFile_Disk additionally covers the file-IO + extension
// dispatch path. The body is written to a temp file with a random extension
// to flip between YAML and JSON branches.
func FuzzLoadRulesetFile_Disk(f *testing.F) {
	f.Add([]byte(`{"rules":[{"name":"r","match":"any","pattern":"a","weight":1}]}`), ".json")
	f.Add([]byte("rules:\n  - name: r\n    match: any\n    pattern: a\n    weight: 1\n"), ".yaml")
	f.Add([]byte(""), ".json")
	f.Add([]byte("garbage"), ".yml")

	f.Fuzz(func(t *testing.T, body []byte, ext string) {
		switch ext {
		case ".json", ".yaml", ".yml":
		default:
			ext = ".json"
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "ruleset"+ext)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
		_, _ = LoadRulesetFile(path) // must not panic regardless of body
	})
}

// FuzzExtractPatternTokens guarantees the pattern tokenizer is total
// (no panic) and never produces empty tokens.
func FuzzExtractPatternTokens(f *testing.F) {
	f.Add("(?i)(union|select|drop)")
	f.Add("()")
	f.Add("|||")
	f.Add("")
	f.Add("(?I)x")

	f.Fuzz(func(t *testing.T, pattern string) {
		for _, tok := range extractPatternTokens(pattern) {
			if tok == "" {
				t.Fatalf("empty token from pattern=%q", pattern)
			}
		}
	})
}
