package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Xwudao/neter/pkg/utils"
)

var migrationFilePattern = regexp.MustCompile(`^(\d+)_`)

// CreatedMigration reports the paths written by Create.
type CreatedMigration struct {
	Version  int
	UpPath   string
	DownPath string
}

// Create writes an empty paired migration (<version>_<name>.up.sql and
// .down.sql) into dir, using the next zero-padded version after the highest
// existing migration. The directory is created when missing.
func Create(dir, name string) (*CreatedMigration, error) {
	slug := slugify(name)
	if slug == "" {
		return nil, fmt.Errorf("migration name is required (e.g. nr migrate new add_users)")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create migration directory: %w", err)
	}

	version, err := nextVersion(dir)
	if err != nil {
		return nil, err
	}

	base := fmt.Sprintf("%06d_%s", version, slug)
	upPath := filepath.Join(dir, base+".up.sql")
	downPath := filepath.Join(dir, base+".down.sql")

	for _, p := range []string{upPath, downPath} {
		if _, err := os.Stat(p); err == nil {
			return nil, fmt.Errorf("migration already exists: %s", p)
		}
	}

	upBody := fmt.Sprintf("-- %s\n-- write the forward migration here\n", base+".up.sql")
	downBody := fmt.Sprintf("-- %s\n-- write the rollback migration here\n", base+".down.sql")

	if err := utils.WriteFileAtomic(upPath, []byte(upBody), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", upPath, err)
	}
	if err := utils.WriteFileAtomic(downPath, []byte(downBody), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", downPath, err)
	}

	return &CreatedMigration{Version: version, UpPath: upPath, DownPath: downPath}, nil
}

// nextVersion returns the highest existing migration version + 1.
func nextVersion(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("read migration directory: %w", err)
	}

	highest := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}
		v, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if v > highest {
			highest = v
		}
	}
	return highest + 1, nil
}

// slugify converts a free-form name into a lower_snake_case filename segment.
func slugify(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case r == '_' || r == '-' || r == ' ' || r == '.':
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
