package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebPublishEnv(t *testing.T) {
	t.Setenv("PUBLISH_TEST_BUCKET", "personal-assets")
	cfg := &NeterConfig{Build: BuildConfig{Frontend: FrontendBuildConfig{Publish: WebPublishConfig{
		Enabled: true, Bucket: "${PUBLISH_TEST_BUCKET}", PublicURL: "https://cdn.example.com",
		AccessKeyIDEnv: "CUSTOM_KEY", SecretAccessKeyEnv: "CUSTOM_SECRET",
	}}}}
	for _, tc := range []struct {
		name        string
		cfg         *NeterConfig
		override    *bool
		wantEnabled bool
	}{
		{name: "absent config defaults off"},
		{name: "configured", cfg: cfg, wantEnabled: true},
		{name: "explicit false wins", cfg: cfg, override: new(false)},
		{name: "explicit true", override: new(true), wantEnabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, err := tc.cfg.WebPublishEnv(tc.override)
			if err != nil {
				t.Fatal(err)
			}
			if len(env) != 1 {
				t.Fatalf("env = %v", env)
			}
			data, ok := strings.CutPrefix(env[0], "NETER_WEB_PUBLISH_CONFIG=")
			if !ok {
				t.Fatalf("unexpected env name: %v", env)
			}
			var got WebPublishConfig
			if err := json.Unmarshal([]byte(data), &got); err != nil {
				t.Fatal(err)
			}
			if got.Enabled != tc.wantEnabled {
				t.Fatalf("enabled = %v", got.Enabled)
			}
			if !tc.wantEnabled && data != `{"enabled":false}` {
				t.Fatalf("disabled config leaked personal settings: %s", data)
			}
			if tc.cfg != nil && tc.wantEnabled {
				if got.Bucket != "personal-assets" || got.AccessKeyIDEnv != "CUSTOM_KEY" || got.SecretAccessKeyEnv != "CUSTOM_SECRET" {
					t.Fatalf("unexpected settings: %+v", got)
				}
			}
		})
	}
	if cfg.Build.Frontend.Publish.Bucket != "${PUBLISH_TEST_BUCKET}" {
		t.Fatal("mutated original configuration")
	}
}

func TestParseWebPublishConfig(t *testing.T) {
	file := filepath.Join(t.TempDir(), "neter.yml")
	if err := os.WriteFile(file, []byte(`build:
  frontend:
    api_base_url: https://api.example.com
    publish:
      enabled: true
      endpoint: https://account.r2.cloudflarestorage.com
      region: auto
      bucket: assets
      public_url: https://cdn.example.com
      prefix: app/releases/version
      access_key_id_env: CUSTOM_ID
      secret_access_key_env: CUSTOM_SECRET
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseNeterFile(file)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.Build.Frontend.Publish
	if !got.Enabled || got.Region != "auto" || got.Endpoint == "" || got.Bucket != "assets" || got.PublicURL != "https://cdn.example.com" || got.Prefix != "app/releases/version" || got.AccessKeyIDEnv != "CUSTOM_ID" || got.SecretAccessKeyEnv != "CUSTOM_SECRET" {
		t.Fatalf("publish settings not parsed: %+v", got)
	}
	if cfg.Build.Frontend.APIBaseURL != "https://api.example.com" {
		t.Fatalf("api_base_url not parsed: %q", cfg.Build.Frontend.APIBaseURL)
	}
}

func TestFrontendAPIBaseEnv(t *testing.T) {
	t.Setenv("API_BASE_TEST_URL", "https://api.from-env.example.com")
	for _, tc := range []struct {
		name string
		cfg  *NeterConfig
		want []string
	}{
		{name: "absent config exports nothing"},
		{name: "empty value exports nothing", cfg: &NeterConfig{}},
		{
			name: "configured value is exported",
			cfg:  &NeterConfig{Build: BuildConfig{Frontend: FrontendBuildConfig{APIBaseURL: "https://api.example.com"}}},
			want: []string{"VITE_API_BASE_URL=https://api.example.com"},
		},
		{
			name: "expands shell variables",
			cfg:  &NeterConfig{Build: BuildConfig{Frontend: FrontendBuildConfig{APIBaseURL: "${API_BASE_TEST_URL}"}}},
			want: []string{"VITE_API_BASE_URL=https://api.from-env.example.com"},
		},
		{
			name: "unset placeholder exports nothing",
			cfg:  &NeterConfig{Build: BuildConfig{Frontend: FrontendBuildConfig{APIBaseURL: "${API_BASE_TEST_UNSET}"}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.FrontendAPIBaseEnv()
			if len(got) != len(tc.want) {
				t.Fatalf("env = %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("env = %v, want %v", got, tc.want)
				}
			}
		})
	}
}
