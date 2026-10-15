package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildWebAssetsPublishFailureDoesNotCopy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires POSIX")
	}
	// Child-specific settings must override an inherited personal enablement.
	t.Setenv("NETER_WEB_PUBLISH_CONFIG", `{"enabled":true}`)
	t.Chdir(t.TempDir())
	for _, dir := range []string{"web-v2/dist", "assets/dist"} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile("assets/dist/index.html", []byte("existing standalone HTML"), 0o600); err != nil {
		t.Fatal(err)
	}
	pm, err := filepath.Abs("fake-pm")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate Vite writing CDN HTML and then failing in the publishing hook.
	script := "#!/bin/sh\nif [ \"$1\" = install ]; then exit 0; fi\nprintf '%s' \"$NETER_WEB_PUBLISH_CONFIG\" > received-env\necho CDN > dist/index.html\nexit 1\n"
	if err := os.WriteFile(pm, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	publishEnv := `NETER_WEB_PUBLISH_CONFIG={"enabled":true}`
	if err := buildWebAssets("web-v2", pm, false, publishEnv); err == nil {
		t.Fatal("expected build failure")
	}
	data, err := os.ReadFile("assets/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing standalone HTML" {
		t.Fatalf("failed publish overwrote embedded HTML: %s", data)
	}
	data, err = os.ReadFile("web-v2/received-env")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"enabled":true}` {
		t.Fatalf("publish env not forwarded: %s", data)
	}

	// A later standalone build must replace the previous CDN output.
	script = "#!/bin/sh\nif [ \"$1\" = install ]; then exit 0; fi\nprintf '%s' \"$NETER_WEB_PUBLISH_CONFIG\" > dist/index.html\n"
	if err := os.WriteFile(pm, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := buildWebAssets("web-v2", pm, false, `NETER_WEB_PUBLISH_CONFIG={"enabled":false}`); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile("assets/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"enabled":false`) || strings.Contains(string(data), "CDN") {
		t.Fatalf("stale CDN output: %s", data)
	}
}
