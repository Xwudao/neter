package core

import (
	"os"
	"path/filepath"
)

// ProjectKind classifies a neter project by its persistence layer so the CLI
// can support both the legacy Ent/MySQL stack and the new PostgreSQL + pgx +
// sqlc stack.
type ProjectKind string

const (
	// ProjectKindUnknown means neither the Ent nor the sqlc markers were found.
	ProjectKindUnknown ProjectKind = "unknown"
	// ProjectKindSQLC is the new template: sqlc.yaml + db/query + db/migrations.
	ProjectKindSQLC ProjectKind = "sqlc"
	// ProjectKindEnt is the legacy template: internal/data/ent/schema.
	ProjectKindEnt ProjectKind = "ent"
)

// IsSQLC reports whether the project uses the new sqlc persistence layer.
func (k ProjectKind) IsSQLC() bool { return k == ProjectKindSQLC }

// IsEnt reports whether the project uses the legacy Ent persistence layer.
func (k ProjectKind) IsEnt() bool { return k == ProjectKindEnt }

// DetectProjectKind classifies the project rooted at root. sqlc wins when both
// marker sets are present so a project that is mid-migration is treated as the
// new stack (the Ent directory may still be lingering).
func DetectProjectKind(root string) ProjectKind {
	if fileExists(filepath.Join(root, "sqlc.yaml")) || dirExists(filepath.Join(root, "db", "query")) {
		return ProjectKindSQLC
	}
	if dirExists(filepath.Join(root, "internal", "data", "ent", "schema")) {
		return ProjectKindEnt
	}
	return ProjectKindUnknown
}

// DetectCurrentProjectKind classifies the project containing the current
// working directory, walking up to the go.mod boundary.
func DetectCurrentProjectKind() ProjectKind {
	dir, err := os.Getwd()
	if err != nil {
		return ProjectKindUnknown
	}
	for range 8 {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return DetectProjectKind(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return DetectProjectKind(dir)
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
