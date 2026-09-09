package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectKind(t *testing.T) {
	t.Run("sqlc via sqlc.yaml", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "sqlc.yaml"), "version: \"2\"")
		if got := DetectProjectKind(root); got != ProjectKindSQLC {
			t.Fatalf("got %q, want %q", got, ProjectKindSQLC)
		}
	})

	t.Run("sqlc via db/query", func(t *testing.T) {
		root := t.TempDir()
		mkdir(t, filepath.Join(root, "db", "query"))
		if got := DetectProjectKind(root); got != ProjectKindSQLC {
			t.Fatalf("got %q, want %q", got, ProjectKindSQLC)
		}
	})

	t.Run("ent", func(t *testing.T) {
		root := t.TempDir()
		mkdir(t, filepath.Join(root, "internal", "data", "ent", "schema"))
		if got := DetectProjectKind(root); got != ProjectKindEnt {
			t.Fatalf("got %q, want %q", got, ProjectKindEnt)
		}
	})

	t.Run("sqlc wins when both exist", func(t *testing.T) {
		root := t.TempDir()
		mkdir(t, filepath.Join(root, "internal", "data", "ent", "schema"))
		writeFile(t, filepath.Join(root, "sqlc.yaml"), "version: \"2\"")
		if got := DetectProjectKind(root); got != ProjectKindSQLC {
			t.Fatalf("got %q, want %q", got, ProjectKindSQLC)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		if got := DetectProjectKind(t.TempDir()); got != ProjectKindUnknown {
			t.Fatalf("got %q, want %q", got, ProjectKindUnknown)
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
