// Package migrate implements `nr migrate` database migrations for neter
// projects. It embeds golang-migrate so a project can create paired
// .up.sql/.down.sql files and apply them to a real database without having to
// compile the application first.
//
// The same package serves both persistence stacks:
//   - new projects: PostgreSQL + pgx + sqlc, migrations under db/migrations
//   - legacy projects: MySQL + Ent, when a config.yml/db dialect is present
package migrate

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultDir is the migration directory used by the new sqlc template.
	DefaultDir = "db/migrations"
	// DefaultConfigFile is the runtime config read to build the DSN.
	DefaultConfigFile = "config.yml"
	// EnvDSN is the highest-priority DSN environment variable.
	EnvDSN = "NETER_DSN"
)

// Options describes a migration invocation resolved against a project root.
type Options struct {
	Root       string // absolute project root (directory containing go.mod)
	Dir        string // absolute directory holding *.up.sql / *.down.sql
	DSN        string // explicit DSN; empty falls back to env then config.yml
	ConfigFile string // absolute config.yml path used for DSN fallback
	Dialect    string // explicit dialect override (postgres|mysql)
}

// DBConfig mirrors the db: block shared by the old and new config.yml files.
type DBConfig struct {
	Dialect  string `yaml:"dialect"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type fileConfig struct {
	DB DBConfig `yaml:"db"`
}

// NewOptions resolves root/dir/config paths. Empty values fall back to the
// current directory, DefaultDir and DefaultConfigFile respectively.
func NewOptions(root, dir, dsn, configFile, dialect string) (*Options, error) {
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		root = cwd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	if dir == "" {
		dir = DefaultDir
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(absRoot, dir)
	}

	if configFile == "" {
		configFile = filepath.Join(absRoot, DefaultConfigFile)
	} else if !filepath.IsAbs(configFile) {
		configFile = filepath.Join(absRoot, configFile)
	}

	return &Options{
		Root:       absRoot,
		Dir:        filepath.Clean(dir),
		DSN:        dsn,
		ConfigFile: configFile,
		Dialect:    dialect,
	}, nil
}

// ResolveDSN returns the DSN and the golang-migrate database driver name.
// Priority: --dsn flag > NETER_DSN/DATABASE_URL > config.yml db block.
func (o *Options) ResolveDSN() (dsn string, driver string, err error) {
	if o.DSN != "" {
		return o.DSN, driverFor(o.Dialect, o.DSN), nil
	}
	for _, env := range []string{EnvDSN, "DATABASE_URL"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v, driverFor(o.Dialect, v), nil
		}
	}

	cfg, err := o.loadConfig()
	if err != nil {
		return "", "", err
	}
	dialect := o.Dialect
	if dialect == "" {
		dialect = cfg.DB.Dialect
	}
	dsn, err = buildDSN(dialect, cfg.DB)
	if err != nil {
		return "", "", err
	}
	return dsn, driverFor(dialect, dsn), nil
}

func (o *Options) loadConfig() (*fileConfig, error) {
	data, err := os.ReadFile(o.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w (pass --dsn or set %s to skip)", o.ConfigFile, err, EnvDSN)
	}
	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", o.ConfigFile, err)
	}
	return &cfg, nil
}

// buildDSN builds a golang-migrate compatible DSN for the given dialect.
func buildDSN(dialect string, c DBConfig) (string, error) {
	if c.Host == "" {
		return "", fmt.Errorf("db.host is empty in config.yml (pass --dsn or set %s)", EnvDSN)
	}

	switch strings.ToLower(dialect) {
	case "mysql":
		port := c.Port
		if port == 0 {
			port = 3306
		}
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
			url.QueryEscape(c.Username), url.QueryEscape(c.Password), c.Host, port, c.Database), nil
	default: // postgres is the new template default
		port := c.Port
		if port == 0 {
			port = 5432
		}
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			url.QueryEscape(c.Username), url.QueryEscape(c.Password), c.Host, port, url.PathEscape(c.Database)), nil
	}
}

// driverFor picks a golang-migrate database driver. The DSN scheme wins over
// the configured dialect so an explicit --dsn always behaves as expected.
func driverFor(dialect, dsn string) string {
	switch {
	case strings.HasPrefix(dsn, "mysql://"):
		return "mysql"
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return "postgres"
	case strings.EqualFold(dialect, "mysql"):
		return "mysql"
	default:
		return "postgres"
	}
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
