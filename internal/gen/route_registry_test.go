package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateRouteRegistryAddsOnlyTheNewRegistryEntry(t *testing.T) {
	root := t.TempDir()
	routesDir := filepath.Join(root, "internal", "routes")
	if err := os.MkdirAll(filepath.Join(routesDir, "v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := `package routes

import v1 "example.com/project/internal/routes/v1"

type Registrar interface { Register() }
type RouteRegistry []Registrar

func NewRouteRegistry(userRoute *v1.UserRoute) RouteRegistry {
	return RouteRegistry{userRoute}
}
`
	registryPath := filepath.Join(routesDir, "registry.go")
	if err := os.WriteFile(registryPath, []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}

	g := &Generator{
		RootPath:        filepath.Join(routesDir, "v1"),
		PackageName:     "v1",
		ModName:         "example.com/project",
		StructRouteName: "HealthRoute",
	}
	if !g.hasRouteRegistry() {
		t.Fatal("expected opt-in registry to be detected")
	}
	if err := g.updateRouteRegistry(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	source := string(got)
	if !strings.Contains(source, "healthRoute *v1.HealthRoute") {
		t.Fatalf("new route parameter missing:\n%s", source)
	}
	if !strings.Contains(source, "RouteRegistry{userRoute, healthRoute}") {
		t.Fatalf("new route registration missing:\n%s", source)
	}
}

func TestHasRouteRegistryLeavesLegacyProjectsOnTheExistingPath(t *testing.T) {
	g := &Generator{RootPath: filepath.Join(t.TempDir(), "internal", "routes", "v1")}
	if g.hasRouteRegistry() {
		t.Fatal("legacy project unexpectedly opted into RouteRegistry")
	}
}
