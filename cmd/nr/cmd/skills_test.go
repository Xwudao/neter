package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Xwudao/neter/internal/core"
)

func TestResolveSkillKind(t *testing.T) {
	root := t.TempDir()

	if got, err := resolveSkillKind(root, "auto"); err != nil || got != core.ProjectKindUnknown {
		t.Fatalf("auto on empty dir = %v, %v", got, err)
	}

	if err := os.WriteFile(filepath.Join(root, "sqlc.yaml"), []byte("version: \"2\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := resolveSkillKind(root, "auto"); got != core.ProjectKindSQLC {
		t.Fatalf("auto with sqlc.yaml = %v", got)
	}

	if got, _ := resolveSkillKind(root, "ent"); got != core.ProjectKindEnt {
		t.Fatalf("forced ent = %v", got)
	}

	if _, err := resolveSkillKind(root, "bogus"); err == nil {
		t.Fatal("unknown --kind must return an error")
	}
}
