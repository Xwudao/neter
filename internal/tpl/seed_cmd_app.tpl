package cmd_app

import (
	"context"

	"{{.ModName}}/internal/seed"
	"{{.ModName}}/internal/system"
)

type SeedApp struct {
	ctx      context.Context
	registry *seed.Registry
}

func NewSeedApp(app *system.AppContext, registry *seed.Registry) *SeedApp {
	return &SeedApp{ctx: app.Ctx, registry: registry}
}

func (a *SeedApp) Names() []string {
	return a.registry.Names()
}

func (a *SeedApp) Run(names []string, options seed.Options) ([]string, error) {
	return a.registry.Run(a.ctx, names, options)
}
