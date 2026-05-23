package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LdflagVar represents a single ldflags variable injection.
type LdflagVar struct {
	Package string            `yaml:"package"`
	Vars    map[string]string `yaml:"vars"`
}

// NeterConfig represents the neter.yml project build configuration.
type NeterConfig struct {
	Ldflags []LdflagVar `yaml:"ldflags"`
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
			return nil, fmt.Errorf("neter.yml not found")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, fmt.Errorf("neter.yml not found in project tree")
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

// BuildLdflags constructs the -ldflags string for go build.
// It resolves ${ENV_VAR} placeholders in variable values from the environment.
func (c *NeterConfig) BuildLdflags() string {
	if len(c.Ldflags) == 0 {
		return ""
	}

	var parts []string
	for _, lf := range c.Ldflags {
		for varName, varValue := range lf.Vars {
			resolved := os.ExpandEnv(varValue)
			parts = append(parts, fmt.Sprintf("-X '%s.%s=%s'", lf.Package, varName, resolved))
		}
	}

	return strings.Join(parts, " ")
}
