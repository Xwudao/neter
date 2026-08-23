package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaleGeneratedTypeScriptFilesIgnoresHandWrittenFiles(t *testing.T) {
	dir := t.TempDir()
	generated := filepath.Join(dir, "old.gen.ts")
	if err := os.WriteFile(generated, []byte(generatedTypeScriptHeader+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manual.gen.ts"), []byte("export const manual = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, err := staleGeneratedTypeScriptFiles(dir, map[string]string{"current.gen.ts": ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0] != "old.gen.ts" {
		t.Fatalf("stale = %#v, want only old.gen.ts", stale)
	}
	if err := removeStaleGeneratedTypeScriptFiles(dir, map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(generated); !os.IsNotExist(err) {
		t.Fatalf("generated stale file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "manual.gen.ts")); err != nil {
		t.Fatalf("hand-written file was removed: %v", err)
	}
}
