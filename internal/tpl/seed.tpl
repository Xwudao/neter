package seed

import "context"

// {{.StructSeedName}} initializes the {{.ToKebab .Name}} data set.
// Add dependencies deliberately through NewRegistry rather than writing
// directly from a command handler.
type {{.StructSeedName}} struct{}

func New{{.StructSeedName}}() *{{.StructSeedName}} {
	return &{{.StructSeedName}}{}
}

func (s *{{.StructSeedName}}) Name() string {
	return "{{.ToKebab .Name}}"
}

func (s *{{.StructSeedName}}) Run(_ context.Context, options Options) error {
	if options.DryRun {
		return nil
	}
	// TODO: add idempotent initialization. Do not overwrite operator-managed
	// data unless options.Force is explicitly handled here.
	return nil
}
