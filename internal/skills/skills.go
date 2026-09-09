package skills

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Xwudao/neter/internal/core"
)

//go:embed templates/*.md.tmpl
var templates embed.FS

const (
	// Dir is the agent-skills directory, relative to the project root.
	Dir = ".agents/skills"
	// DefaultName is the generated skill's name and directory.
	DefaultName = "neter"
)

// Data is the template payload for a generated skill.
type Data struct {
	Name string
	Root string
	Kind core.ProjectKind
}

// IsSQLC reports whether the project uses PostgreSQL + pgx + sqlc.
func (d Data) IsSQLC() bool { return d.Kind.IsSQLC() }

// IsEnt reports whether the project uses the legacy Ent + MySQL stack.
func (d Data) IsEnt() bool { return d.Kind.IsEnt() }

// IsUnknown reports whether the persistence stack could not be detected.
func (d Data) IsUnknown() bool { return !d.Kind.IsSQLC() && !d.Kind.IsEnt() }

// Render returns the SKILL.md content for the given project kind.
func Render(name, root string, kind core.ProjectKind) (string, error) {
	if name == "" {
		name = DefaultName
	}

	tpl, err := template.New("skill").ParseFS(templates, "templates/neter.md.tmpl")
	if err != nil {
		return "", fmt.Errorf("parse skill template: %w", err)
	}

	var buf strings.Builder
	data := Data{Name: name, Root: filepath.ToSlash(root), Kind: kind}
	if err := tpl.ExecuteTemplate(&buf, "neter.md.tmpl", data); err != nil {
		return "", fmt.Errorf("render skill template: %w", err)
	}
	return buf.String(), nil
}

// Write renders the skill and writes it to
// <root>/.agents/skills/<name>/SKILL.md, overwriting any previous copy.
func Write(root, name string, kind core.ProjectKind) (string, error) {
	if name == "" {
		name = DefaultName
	}

	content, err := Render(name, root, kind)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(root, Dir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}

	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// ResolveRoot returns the project root. When dir is empty it walks up from the
// working directory to the nearest go.mod, falling back to the working
// directory when no module is found.
func ResolveRoot(dir string) (string, error) {
	if dir != "" {
		return filepath.Abs(dir)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cur := cwd
	for range 8 {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return cwd, nil
}
