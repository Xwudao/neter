package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFrontendOptionsDefault(t *testing.T) {
	// No neter.yml anywhere above: fall back to the defaults.
	webDir, pm := resolveFrontendOptions("pnpm", false)
	if webDir != "web" {
		t.Fatalf("expected default web dir %q, got %q", "web", webDir)
	}
	if pm != "pnpm" {
		t.Fatalf("expected pm to stay %q, got %q", "pnpm", pm)
	}
}

func TestResolveFrontendOptionsFromConfig(t *testing.T) {
	dir := t.TempDir()
	neterYml := `dev:
  frontend:
    dir: "web-v2"
    pm: "npm"
`
	if err := os.WriteFile(filepath.Join(dir, "neter.yml"), []byte(neterYml), 0o644); err != nil {
		t.Fatalf("write neter.yml: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Without an explicit pm flag, both dir and pm come from neter.yml.
	webDir, pm := resolveFrontendOptions("pnpm", false)
	if webDir != "web-v2" {
		t.Fatalf("expected web dir from config %q, got %q", "web-v2", webDir)
	}
	if pm != "npm" {
		t.Fatalf("expected pm from config %q, got %q", "npm", pm)
	}

	// An explicitly passed pm flag takes precedence over the config.
	webDir, pm = resolveFrontendOptions("yarn", true)
	if webDir != "web-v2" {
		t.Fatalf("expected web dir from config %q, got %q", "web-v2", webDir)
	}
	if pm != "yarn" {
		t.Fatalf("expected pm from flag %q, got %q", "yarn", pm)
	}
}
