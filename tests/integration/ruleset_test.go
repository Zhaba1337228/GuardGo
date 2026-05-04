package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	guardgo "github.com/Zhaba1337228/GuardGo"
)

func TestRulesetManagerReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	initial := []byte("rules:\n  - name: SQLi\n    match: query\n    pattern: \"(?i)(select|union)\"\n    weight: 10\n")
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := guardgo.NewRulesetManager(path)
	if err != nil {
		t.Fatal(err)
	}
	if manager.Current() == nil || manager.Current().Query == nil {
		t.Fatalf("expected compiled query matcher after initial load")
	}
	if len(manager.Current().Query.FindAll("q=SELECT+1")) == 0 {
		t.Fatalf("expected query matcher to find initial token")
	}

	updated := []byte("rules:\n  - name: PathProbe\n    match: path\n    pattern: \"admin\"\n    weight: 7\n")
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Reload(); err != nil {
		t.Fatal(err)
	}
	if manager.Current() == nil || manager.Current().Path == nil {
		t.Fatalf("expected compiled path matcher after reload")
	}
	if len(manager.Current().Path.FindAll("/admin/login")) == 0 {
		t.Fatalf("expected path matcher to find updated token")
	}
}
