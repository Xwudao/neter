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

func TestDeployConfigValidate(t *testing.T) {
	cfg := DeployConfig{
		Alias:           "prod-app",
		RemoteUploadDir: "/srv/myapp",
		RemoteScript:    "/srv/myapp/deploy.sh",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestDeployConfigValidateRequiresFields(t *testing.T) {
	testCases := []struct {
		name string
		cfg  DeployConfig
	}{
		{
			name: "missing alias",
			cfg: DeployConfig{
				RemoteUploadDir: "/srv/myapp",
				RemoteScript:    "/srv/myapp/deploy.sh",
			},
		},
		{
			name: "missing remote upload dir",
			cfg: DeployConfig{
				Alias:        "prod-app",
				RemoteScript: "/srv/myapp/deploy.sh",
			},
		},
		{
			name: "missing remote script",
			cfg: DeployConfig{
				Alias:           "prod-app",
				RemoteUploadDir: "/srv/myapp",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); err == nil {
				t.Fatalf("Validate() error = nil, want error")
			}
		})
	}
}
