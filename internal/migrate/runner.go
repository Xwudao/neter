package migrate

import (
	"errors"
	"fmt"
	"path/filepath"

	gomigrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Status is the migration state reported after an operation.
type Status struct {
	Version    uint
	Dirty      bool
	HasVersion bool // false when no migration has been applied yet
}

// Up applies pending migrations. steps > 0 applies at most that many; steps <= 0
// applies everything pending.
func (o *Options) Up(steps int) (Status, error) {
	m, driver, err := o.newMigrator()
	if err != nil {
		return Status{}, err
	}
	defer closeQuietly(m)

	if steps > 0 {
		err = m.Steps(steps)
	} else {
		err = m.Up()
	}
	if err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return statusOf(m), fmt.Errorf("migrate up (%s): %w", driver, err)
	}
	return statusOf(m), nil
}

// Down rolls back migrations. all=true rolls back every applied migration;
// otherwise steps (default 1) migrations are rolled back.
func (o *Options) Down(steps int, all bool) (Status, error) {
	m, driver, err := o.newMigrator()
	if err != nil {
		return Status{}, err
	}
	defer closeQuietly(m)

	if all {
		err = m.Down()
	} else {
		if steps <= 0 {
			steps = 1
		}
		err = m.Steps(-steps)
	}
	if err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return statusOf(m), fmt.Errorf("migrate down (%s): %w", driver, err)
	}
	return statusOf(m), nil
}

// Status returns the current version and dirty flag. A dirty database is
// reported via Status.Dirty rather than an error so callers can print guidance.
func (o *Options) Status() (Status, error) {
	m, _, err := o.newMigrator()
	if err != nil {
		return Status{}, err
	}
	defer closeQuietly(m)
	return statusOf(m), nil
}

// Force sets the schema version without running migrations, clearing the dirty
// flag. Use it to recover from a failed migration.
func (o *Options) Force(version int) (Status, error) {
	m, driver, err := o.newMigrator()
	if err != nil {
		return Status{}, err
	}
	defer closeQuietly(m)

	if err := m.Force(version); err != nil {
		return statusOf(m), fmt.Errorf("force version %d (%s): %w", version, driver, err)
	}
	return statusOf(m), nil
}

func (o *Options) newMigrator() (*gomigrate.Migrate, string, error) {
	if !dirExists(o.Dir) {
		return nil, "", fmt.Errorf("migration directory not found: %s (create one with `nr migrate new <name>`)", o.Dir)
	}

	dsn, driver, err := o.ResolveDSN()
	if err != nil {
		return nil, "", err
	}

	sourceURL := "file://" + filepath.ToSlash(o.Dir)
	m, err := gomigrate.New(sourceURL, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("open migrator (%s): %w", driver, err)
	}
	return m, driver, nil
}

func statusOf(m *gomigrate.Migrate) Status {
	version, dirty, err := m.Version()
	if err != nil {
		// ErrNilVersion means no migration has been applied yet.
		return Status{}
	}
	return Status{Version: version, Dirty: dirty, HasVersion: true}
}

func closeQuietly(m *gomigrate.Migrate) {
	_, _ = m.Close()
}
