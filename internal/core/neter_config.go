package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var ErrNeterConfigNotFound = errors.New("neter.yml not found")

// LdflagVar represents a single ldflags variable injection.
type LdflagVar struct {
	Package string            `yaml:"package"`
	Vars    map[string]string `yaml:"vars"`
}

type DevConfig struct {
	Backend  DevBackendConfig  `yaml:"backend"`
	Frontend DevFrontendConfig `yaml:"frontend"`
}

type DevBackendConfig struct {
	Cmd string `yaml:"cmd"`
}

type DevFrontendConfig struct {
	Dir string `yaml:"dir"`
	Pm  string `yaml:"pm"`
	Cmd string `yaml:"cmd"`
}

type HooksConfig struct {
	Enabled bool             `yaml:"enabled"`
	Items   []HookItemConfig `yaml:"items"`
}

type HookItemConfig struct {
	Event   string             `yaml:"event"`
	Action  string             `yaml:"action"`
	Depends *HookDependsConfig `yaml:"depends,omitempty"`
}

type HookDependsConfig struct {
	Flags []string `yaml:"flags"`
}

// NeterConfig represents the neter.yml project build configuration.
type NeterConfig struct {
	Ldflags []LdflagVar `yaml:"ldflags"`
	Dev     DevConfig   `yaml:"dev"`
	Hooks   HooksConfig `yaml:"hooks"`
}

// LoadNeterConfig reads and parses the neter.yml file from the project root.
// It walks up the directory tree until it finds a neter.yml file or go.mod.
func LoadNeterConfig() (*NeterConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	for range 10 {
		p := filepath.Join(dir, "neter.yml")
		if _, statErr := os.Stat(p); statErr == nil {
			return parseNeterFile(p)
		}
		// Stop at go.mod boundary
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return nil, ErrNeterConfigNotFound
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, fmt.Errorf("%w in project tree", ErrNeterConfigNotFound)
}

func LoadOptionalNeterConfig() (*NeterConfig, error) {
	cfg, err := LoadNeterConfig()
	if err != nil {
		if errors.Is(err, ErrNeterConfigNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

func parseNeterFile(path string) (*NeterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read neter.yml: %w", err)
	}

	var cfg NeterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse neter.yml: %w", err)
	}

	return &cfg, nil
}

func DefaultDevConfig() DevConfig {
	return DevConfig{
		Backend: DevBackendConfig{
			Cmd: "",
		},
		Frontend: DevFrontendConfig{
			Dir: "web",
			Pm:  "pnpm",
			Cmd: "run dev",
		},
	}
}

func (c *NeterConfig) EffectiveDevConfig() DevConfig {
	cfg := DefaultDevConfig()
	if c == nil {
		return cfg
	}

	if c.Dev.Backend.Cmd != "" {
		cfg.Backend.Cmd = c.Dev.Backend.Cmd
	}
	if c.Dev.Frontend.Dir != "" {
		cfg.Frontend.Dir = c.Dev.Frontend.Dir
	}
	if c.Dev.Frontend.Pm != "" {
		cfg.Frontend.Pm = c.Dev.Frontend.Pm
	}
	if c.Dev.Frontend.Cmd != "" {
		cfg.Frontend.Cmd = c.Dev.Frontend.Cmd
	}

	return cfg
}

func ExampleNeterConfigYAML() string {
	return `# Example neter.yml
ldflags:
  - package: main
    vars:
      buildTime: "${datetime}"
      gitHash: "${git_hash}"
      gitTag: "${git_tag}"
      goVersion: "${go_version}"
  - package: your/module/path/internal/version
    vars:
      env: "prod"
      date: "${date}"
      timestampMs: "${timestamp_ms}"

dev:
  backend:
    cmd: "nr run -dr"
  frontend:
    dir: "web"
    pm: "pnpm"
    cmd: "run dev"

hooks:
  enabled: true
  items:
    - event: "on_start"
      action: "scripts/pre_build.sh"
      depends:
        flags: ["--web"]
    - event: "on_stop"
      action: "scripts/cleanup.sh"
`
}

// BuildLdflags constructs the -ldflags string for go build.
// It resolves ${VAR} placeholders in variable values.
// Built-in variables are resolved first, then environment variables as fallback.
//
// Supported built-in variables:
//   - ${date}          current date (YYYY-MM-DD)
//   - ${datetime}      current datetime (YYYY-MM-DD HH:MM:SS)
//   - ${time}          current time (HH:MM:SS)
//   - ${timestamp}     Unix timestamp in seconds
//   - ${timestamp_ms}  Unix timestamp in milliseconds
//   - ${year}          current year
//   - ${month}         current month (01-12)
//   - ${day}           current day (01-31)
//   - ${git_hash}      short git commit hash (7 chars)
//   - ${git_hash_full} full git commit hash
//   - ${git_tag}       latest git tag (from git describe --tags --abbrev=0)
//   - ${go_version}    Go runtime version
func (c *NeterConfig) BuildLdflags() string {
	if len(c.Ldflags) == 0 {
		return ""
	}

	var parts []string
	for _, lf := range c.Ldflags {
		for varName, varValue := range lf.Vars {
			resolved := expandVars(varValue)
			parts = append(parts, fmt.Sprintf("-X '%s.%s=%s'", lf.Package, varName, resolved))
		}
	}

	return strings.Join(parts, " ")
}

// expandVars resolves ${VAR} placeholders in s.
// Built-in variables take precedence over environment variables.
func expandVars(s string) string {
	return os.Expand(s, func(key string) string {
		if v, ok := getBuiltinVar(key); ok {
			return v
		}
		return os.Getenv(key)
	})
}

// getBuiltinVar returns the value of the named built-in variable.
// The second return value indicates whether the name is a known built-in.
func getBuiltinVar(name string) (string, bool) {
	now := time.Now()
	switch name {
	case "date":
		return now.Format("2006-01-02"), true
	case "datetime":
		return now.Format("2006-01-02 15:04:05"), true
	case "time":
		return now.Format("15:04:05"), true
	case "timestamp":
		return strconv.FormatInt(now.Unix(), 10), true
	case "timestamp_ms":
		return strconv.FormatInt(now.UnixMilli(), 10), true
	case "year":
		return strconv.Itoa(now.Year()), true
	case "month":
		return fmt.Sprintf("%02d", now.Month()), true
	case "day":
		return fmt.Sprintf("%02d", now.Day()), true
	case "git_hash":
		hash, err := GetGitHash()
		if err != nil {
			return "", true
		}
		if len(hash) > 7 {
			hash = hash[:7]
		}
		return hash, true
	case "git_hash_full":
		hash, err := GetGitHash()
		if err != nil {
			return "", true
		}
		return hash, true
	case "git_tag":
		tag, err := GetGitTag()
		if err != nil {
			return "", true
		}
		return tag, true
	case "go_version":
		return runtime.Version(), true
	default:
		return "", false
	}
}
