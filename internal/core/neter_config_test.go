package core

import "testing"

func TestEffectiveDevConfigDefaults(t *testing.T) {
	cfg := (*NeterConfig)(nil).EffectiveDevConfig()

	if cfg.Backend.Cmd != "" {
		t.Fatalf("expected empty backend cmd, got %q", cfg.Backend.Cmd)
	}
	if cfg.Frontend.Dir != "web" {
		t.Fatalf("expected default frontend dir web, got %q", cfg.Frontend.Dir)
	}
	if cfg.Frontend.Pm != "pnpm" {
		t.Fatalf("expected default frontend pm pnpm, got %q", cfg.Frontend.Pm)
	}
	if cfg.Frontend.Cmd != "run dev" {
		t.Fatalf("expected default frontend cmd `run dev`, got %q", cfg.Frontend.Cmd)
	}
}

func TestEffectiveDevConfigOverrides(t *testing.T) {
	cfg := (&NeterConfig{
		Dev: DevConfig{
			Backend: DevBackendConfig{Cmd: "nr run -dr --dir app/admin"},
			Frontend: DevFrontendConfig{
				Dir: "client",
				Pm:  "bun",
				Cmd: "run start",
			},
		},
	}).EffectiveDevConfig()

	if cfg.Backend.Cmd != "nr run -dr --dir app/admin" {
		t.Fatalf("unexpected backend cmd: %q", cfg.Backend.Cmd)
	}
	if cfg.Frontend.Dir != "client" {
		t.Fatalf("unexpected frontend dir: %q", cfg.Frontend.Dir)
	}
	if cfg.Frontend.Pm != "bun" {
		t.Fatalf("unexpected frontend pm: %q", cfg.Frontend.Pm)
	}
	if cfg.Frontend.Cmd != "run start" {
		t.Fatalf("unexpected frontend cmd: %q", cfg.Frontend.Cmd)
	}
}
