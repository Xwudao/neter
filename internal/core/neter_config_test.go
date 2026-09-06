package core

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

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

func TestParseBuildConfigYAML(t *testing.T) {
	var cfg NeterConfig
	src := `build:
  tags:
    - prod
    - netpoll
  cgo: false
`
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal error = %v", err)
	}

	if !reflect.DeepEqual(cfg.Build.Tags, []string{"prod", "netpoll"}) {
		t.Fatalf("Build.Tags = %v, want [prod netpoll]", cfg.Build.Tags)
	}
	if cfg.Build.Cgo == nil {
		t.Fatalf("Build.Cgo is nil, want non-nil (false)")
	}
	if *cfg.Build.Cgo {
		t.Fatalf("Build.Cgo = true, want false")
	}
	if cfg.Build.StopCopy {
		t.Fatalf("Build.StopCopy = true, want false")
	}
}

func TestParseBuildConfigYAMLStopCopy(t *testing.T) {
	var cfg NeterConfig
	src := `build:
  stop_copy: true
`
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal error = %v", err)
	}
	if !cfg.Build.StopCopy {
		t.Fatalf("Build.StopCopy = false, want true")
	}
	if !cfg.StopCopyWeb() {
		t.Fatalf("StopCopyWeb() = false, want true")
	}
}

func TestParseBuildConfigYAMLDefaults(t *testing.T) {
	var cfg NeterConfig
	if err := yaml.Unmarshal([]byte("ldflags: []\n"), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal error = %v", err)
	}
	if cfg.Build.Cgo != nil {
		t.Fatalf("Build.Cgo = %v, want nil", cfg.Build.Cgo)
	}
	if len(cfg.Build.Tags) != 0 {
		t.Fatalf("Build.Tags = %v, want empty", cfg.Build.Tags)
	}
	if cfg.Build.StopCopy {
		t.Fatalf("Build.StopCopy = true, want false")
	}
}

func TestBuildTags(t *testing.T) {
	tests := []struct {
		name string
		cfg  *NeterConfig
		want string
	}{
		{name: "nil config", cfg: nil, want: ""},
		{name: "no tags", cfg: &NeterConfig{}, want: ""},
		{name: "single tag", cfg: &NeterConfig{Build: BuildConfig{Tags: []string{"prod"}}}, want: "prod"},
		{name: "multiple tags", cfg: &NeterConfig{Build: BuildConfig{Tags: []string{"prod", "netpoll"}}}, want: "prod,netpoll"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.BuildTags(); got != tt.want {
				t.Fatalf("BuildTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoBuildEnv(t *testing.T) {
	noCgo := false
	yesCgo := true

	tests := []struct {
		name string
		cfg  *NeterConfig
		want []string
	}{
		{name: "nil config", cfg: nil, want: nil},
		{name: "cgo unset", cfg: &NeterConfig{}, want: nil},
		{name: "cgo enabled", cfg: &NeterConfig{Build: BuildConfig{Cgo: &yesCgo}}, want: []string{"CGO_ENABLED=1"}},
		{name: "cgo disabled", cfg: &NeterConfig{Build: BuildConfig{Cgo: &noCgo}}, want: []string{"CGO_ENABLED=0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GoBuildEnv(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GoBuildEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStopCopyWeb(t *testing.T) {
	tests := []struct {
		name string
		cfg  *NeterConfig
		want bool
	}{
		{name: "nil config", cfg: nil, want: false},
		{name: "stop_copy unset", cfg: &NeterConfig{}, want: false},
		{name: "stop_copy enabled", cfg: &NeterConfig{Build: BuildConfig{StopCopy: true}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.StopCopyWeb(); got != tt.want {
				t.Fatalf("StopCopyWeb() = %v, want %v", got, tt.want)
			}
		})
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
