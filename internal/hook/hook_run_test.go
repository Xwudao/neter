package hook

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldRunOnPlatform(t *testing.T) {
	testCases := []struct {
		name      string
		goos      string
		platforms []string
		action    string
		want      bool
	}{
		{
			name:   "windows runs bat",
			goos:   "windows",
			action: "scripts\\update_random_factor.bat",
			want:   true,
		},
		{
			name:   "windows skips sh",
			goos:   "windows",
			action: "scripts/update_random_factor.sh",
			want:   false,
		},
		{
			name:   "unix skips bat",
			goos:   "linux",
			action: "scripts\\update_random_factor.bat",
			want:   false,
		},
		{
			name:   "unix skips cmd",
			goos:   "darwin",
			action: "scripts\\update_random_factor.cmd --flag",
			want:   false,
		},
		{
			name:   "unix runs sh",
			goos:   "linux",
			action: "scripts/update_random_factor.sh --flag",
			want:   true,
		},
		{
			name:   "empty action skipped",
			goos:   "linux",
			action: "",
			want:   false,
		},
		{
			name:   "binary command allowed everywhere",
			goos:   "linux",
			action: "go version",
			want:   true,
		},
		// Platforms field tests
		{
			name:      "platforms linux runs on linux",
			goos:      "linux",
			platforms: []string{"linux"},
			action:    "scripts/build.sh",
			want:      true,
		},
		{
			name:      "platforms linux skips on mac",
			goos:      "darwin",
			platforms: []string{"linux"},
			action:    "scripts/build.sh",
			want:      false,
		},
		{
			name:      "platforms linux skips on windows",
			goos:      "windows",
			platforms: []string{"linux"},
			action:    "scripts/build.sh",
			want:      false,
		},
		{
			name:      "platforms macos runs on darwin",
			goos:      "darwin",
			platforms: []string{"macos"},
			action:    "scripts/build.sh",
			want:      true,
		},
		{
			name:      "platforms macos skips on linux",
			goos:      "linux",
			platforms: []string{"macos"},
			action:    "scripts/build.sh",
			want:      false,
		},
		{
			name:      "platforms mac also works",
			goos:      "darwin",
			platforms: []string{"mac"},
			action:    "scripts/build.sh",
			want:      true,
		},
		{
			name:      "platforms windows runs on windows",
			goos:      "windows",
			platforms: []string{"windows"},
			action:    "scripts\\build.bat",
			want:      true,
		},
		{
			name:      "platforms windows skips on linux",
			goos:      "linux",
			platforms: []string{"windows"},
			action:    "scripts\\build.bat",
			want:      false,
		},
		{
			name:      "platforms multiple values - matches one",
			goos:      "darwin",
			platforms: []string{"linux", "macos"},
			action:    "scripts/build.sh",
			want:      true,
		},
		{
			name:      "platforms multiple values - no match",
			goos:      "windows",
			platforms: []string{"linux", "macos"},
			action:    "scripts/build.sh",
			want:      false,
		},
		{
			name:      "empty platforms falls back to extension check",
			goos:      "linux",
			platforms: nil,
			action:    "scripts/build.sh",
			want:      true,
		},
		{
			name:      "empty platforms falls back - bat on linux",
			goos:      "linux",
			platforms: nil,
			action:    "scripts\\build.bat",
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRunOnPlatform(tc.goos, tc.platforms, tc.action)
			if got != tc.want {
				t.Fatalf("shouldRunOnPlatform(%q, %v, %q) = %v, want %v", tc.goos, tc.platforms, tc.action, got, tc.want)
			}
		})
	}
}

func TestLoadConfigFromNeterYAML(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/test\n")
	writeTestFile(t, filepath.Join(dir, "neter.yml"), `hooks:
  enabled: true
  items:
    - event: "on_start"
      action: "echo test"
      depends:
        flags: ["--web"]
`)

	restore := chdirForTest(t, dir)
	defer restore()

	manager := NewHookManager()
	if err := manager.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !manager.loaded {
		t.Fatal("expected config to be loaded from neter.yml")
	}
	if !manager.config.App.Enabled {
		t.Fatal("expected hooks to be enabled")
	}
	if len(manager.config.App.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(manager.config.App.Hooks))
	}
	if manager.config.App.Hooks[0].Event != "on_start" {
		t.Fatalf("expected event on_start, got %q", manager.config.App.Hooks[0].Event)
	}
}

func TestLoadConfigWarnsWhenLegacyHookRunExists(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/test\n")
	writeTestFile(t, filepath.Join(dir, "neter.yml"), `hooks:
  enabled: true
  items:
    - event: "on_start"
      action: "echo test"
`)
	writeTestFile(t, filepath.Join(dir, "hook_run.yml"), `app:
  enabled: true
  hooks:
    - event: "on_start"
      action: "echo old"
`)

	restore := chdirForTest(t, dir)
	defer restore()

	var buf bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(oldWriter)
	defer log.SetFlags(oldFlags)

	manager := NewHookManager()
	if err := manager.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !strings.Contains(buf.String(), "legacy") || !strings.Contains(buf.String(), "neter.yml -> hooks") {
		t.Fatalf("expected migration warning, got %q", buf.String())
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func chdirForTest(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	return func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore Chdir(%q) error = %v", wd, err)
		}
	}
}
