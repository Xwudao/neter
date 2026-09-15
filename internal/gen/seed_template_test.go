package gen

import (
	"go/format"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xwudao/neter/internal/tpl"
)

func TestSeedTemplateRenders(t *testing.T) {
	g := &Generator{Name: "siteConfig", StructSeedName: "SiteConfigSeeder"}
	parsed, err := template.New("seed").Parse(tpl.SeedTpl)
	if err != nil {
		t.Fatal(err)
	}

	var output strings.Builder
	if err := parsed.Execute(&output, g); err != nil {
		t.Fatal(err)
	}
	if _, err := format.Source([]byte(output.String())); err != nil {
		t.Fatalf("rendered seed template is not valid Go: %v\n%s", err, output.String())
	}
	for _, want := range []string{"type SiteConfigSeeder struct{}", `return "site-config"`, "options.DryRun"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("rendered seed missing %q\n%s", want, output.String())
		}
	}
}

func TestGenSeedInfrastructure(t *testing.T) {
	root := t.TempDir()
	graphPath := root + "/internal/cmd_app/graph.go"
	require.NoError(t, os.MkdirAll(root+"/internal/cmd_app", 0o755))
	require.NoError(t, os.WriteFile(graphPath, []byte(`package cmd_app

import (
	"github.com/Xwudao/loom"

	"example.com/app/internal/system"
)

var migrateAppGraph = loom.Graph[*MigrateApp](
	loom.Name("MigrateCmd"),
	loom.Provide(NewMigrateApp),
	loom.Provide(system.NewAppContext),
)
`), 0o644))

	g := &Generator{
		RootPath:               root + "/internal/seed",
		ProjectRoot:            root,
		ModName:                "example.com/app",
		saveSeedCmdFilePath:    root + "/internal/cmd/seed.go",
		saveSeedCmdAppFilePath: root + "/internal/cmd_app/seed_app.go",
		saveSeedGraphFilePath:  graphPath,
		seedRegistryTpl:        tpl.SeedRegistryTpl,
		seedCmdTpl:             tpl.SeedCmdTpl,
		seedCmdAppTpl:          tpl.SeedCmdAppTpl,
	}
	require.NoError(t, g.GenSeedInfrastructure())

	for _, path := range []string{root + "/internal/seed/seed.go", g.saveSeedCmdFilePath, g.saveSeedCmdAppFilePath} {
		assert.FileExists(t, path)
	}
	graphSource, err := os.ReadFile(graphPath)
	require.NoError(t, err)
	assert.Contains(t, string(graphSource), "seed.NewRegistry")
	assert.Contains(t, string(graphSource), `loom.Name("SeedCmd")`)
	// The new graph must join the existing block rather than shadow it.
	assert.Contains(t, string(graphSource), "migrateAppGraph = loom.Graph[*MigrateApp](")
}

func TestUpdateSeedRegistry(t *testing.T) {
	dir := t.TempDir()
	registryPath := dir + "/seed.go"
	const registry = `package seed

type Seeder interface{}
type Registry struct{}

func NewRegistry() *Registry {
	return newRegistry()
}

func newRegistry(...Seeder) *Registry { return &Registry{} }
`
	if err := os.WriteFile(registryPath, []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}

	g := &Generator{
		Name:             "demoData",
		RootPath:         dir,
		StructSeedName:   "DemoDataSeeder",
		saveSeedFilePath: dir + "/demo_data.go",
		seedTpl:          tpl.SeedTpl,
	}
	require.NoError(t, g.GenSeed())
	require.NoError(t, g.updateSeedRegistry())

	updated, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	assert.Contains(t, string(updated), "newRegistry(NewDemoDataSeeder())")
	assert.FileExists(t, g.saveSeedFilePath)
}
