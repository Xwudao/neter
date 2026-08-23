package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateKoanfRewritesImportsAndGoMod(t *testing.T) {
	root := t.TempDir()
	goMod := `module example.com/project

go 1.26

require github.com/knadh/koanf v1.5.0
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	goFile := filepath.Join(root, "internal", "config", "config.go")
	if err := os.MkdirAll(filepath.Dir(goFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goFile, []byte("package config\n\nimport \"github.com/knadh/koanf\"\n\nvar Config *koanf.Koanf\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := planKoanfMigration(root)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.legacy || len(plan.goFiles) != 1 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if err := migrateKoanf(root, true, false, true, "v2.2.2"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"github.com/knadh/koanf/v2"`) {
		t.Fatalf("legacy Go import was not migrated:\n%s", content)
	}
	content, err = os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "github.com/knadh/koanf v1") || !strings.Contains(string(content), "github.com/knadh/koanf/v2 v2.2.2") {
		t.Fatalf("go.mod was not migrated:\n%s", content)
	}
}
