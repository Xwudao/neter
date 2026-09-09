package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Xwudao/neter/internal/core"
)

func TestRenderSQLC(t *testing.T) {
	out, err := Render(DefaultName, "/tmp/shop", core.ProjectKindSQLC)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{
		"name: neter",
		"Detected persistence stack: **sqlc**",
		"--model Order --plural Orders",
		"nr migrate new add_orders",
		"SQL is the source of truth.",
		"data.Queries",
		"nr migrate up --steps 1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sqlc skill missing %q", want)
		}
	}
	if strings.Contains(out, "--ent-name") {
		t.Error("sqlc skill must not contain the Ent section")
	}
}

func TestRenderEnt(t *testing.T) {
	out, err := Render(DefaultName, "/tmp/shop", core.ProjectKindEnt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{
		"Detected persistence stack: **ent**",
		"--with-crud --ent-name",
		"nr gen ent",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ent skill missing %q", want)
		}
	}
	if strings.Contains(out, "SQL is the source of truth.") {
		t.Error("ent skill must not contain the sqlc section")
	}
}

func TestRenderUnknown(t *testing.T) {
	out, err := Render(DefaultName, "/tmp/shop", core.ProjectKindUnknown)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{
		"could not be detected",
		"Do not guess a CRUD model flag.",
		"--model Order --plural Orders",
		"--ent-name Order",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("unknown skill missing %q", want)
		}
	}
}

func TestWriteAndOverwrite(t *testing.T) {
	root := t.TempDir()

	path, err := Write(root, DefaultName, core.ProjectKindSQLC)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	want := filepath.Join(root, Dir, DefaultName, "SKILL.md")
	if path != want {
		t.Fatalf("path = %s, want %s", path, want)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(string(content), "---\nname: neter\n") {
		head := string(content)
		if len(head) > 80 {
			head = head[:80]
		}
		t.Fatalf("unexpected frontmatter:\n%s", head)
	}

	// A second run overwrites the previous file.
	if _, err := Write(root, DefaultName, core.ProjectKindEnt); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	content, _ = os.ReadFile(path)
	if strings.Contains(string(content), "SQL is the source of truth.") {
		t.Error("rewrite should replace the sqlc content with the Ent content")
	}
}

func TestResolveRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "internal", "biz")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	got, err := ResolveRoot("")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRoot = %s, want %s", got, root)
	}
}
