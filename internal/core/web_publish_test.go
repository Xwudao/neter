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
}
