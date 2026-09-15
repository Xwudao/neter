// Package seed contains explicit, idempotent data initialization workflows.
package seed

import (
	"context"
	"fmt"
	"slices"
)

type Options struct {
	DryRun bool
	Force  bool
}

type Seeder interface {
	Name() string
	Run(ctx context.Context, options Options) error
}

type Registry struct {
	seeders map[string]Seeder
	names   []string
}

func NewRegistry() *Registry {
	return newRegistry()
}

func newRegistry(seeders ...Seeder) *Registry {
	registry := &Registry{seeders: make(map[string]Seeder, len(seeders))}
	for _, seeder := range seeders {
		name := seeder.Name()
		if _, exists := registry.seeders[name]; exists {
			panic(fmt.Sprintf("duplicate seeder name: %s", name))
		}
		registry.seeders[name] = seeder
		registry.names = append(registry.names, name)
	}
	slices.Sort(registry.names)
	return registry
}

func (r *Registry) Names() []string {
	return slices.Clone(r.names)
}

func (r *Registry) Run(ctx context.Context, names []string, options Options) ([]string, error) {
	if len(names) == 0 {
		names = r.names
	}

	run := make([]string, 0, len(names))
	for _, name := range names {
		seeder, ok := r.seeders[name]
		if !ok {
			return run, fmt.Errorf("unknown seeder %q", name)
		}
		if err := seeder.Run(ctx, options); err != nil {
			return run, fmt.Errorf("run seeder %q: %w", name, err)
		}
		run = append(run, name)
	}
	return run, nil
}
