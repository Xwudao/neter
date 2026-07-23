package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/Xwudao/neter/internal/tpl"
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

func TestHasRouterRegisterRequiresInjectedRegistry(t *testing.T) {
	root := t.TempDir()
	routesDir := filepath.Join(root, "internal", "routes")
	if err := os.MkdirAll(filepath.Join(routesDir, "v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	registryPath := filepath.Join(routesDir, "registry.go")
	if err := os.WriteFile(registryPath, []byte(`package routes

import "github.com/gin-gonic/gin"

type Registrar interface { Register(router gin.IRouter) }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	g := &Generator{RootPath: filepath.Join(routesDir, "v1")}
	if !g.hasRouterRegister() {
		t.Fatal("expected injected registry to be detected")
	}
}

func TestRouteTemplateKeepsEngineForLegacyGeneration(t *testing.T) {
	g := &Generator{
		PackageName:     "v1",
		ModName:         "example.com/project",
		Name:            "health",
		StructRouteName: "HealthRoute",
	}

	parsed, err := template.New("route").Parse(tpl.RouteTpl)
	if err != nil {
		t.Fatal(err)
	}

	var rendered strings.Builder
	if err := parsed.Execute(&rendered, g); err != nil {
		t.Fatal(err)
	}

	source := rendered.String()
	if !strings.Contains(source, "g    *gin.Engine") || !strings.Contains(source, "func (r *HealthRoute) Reg()") {
		t.Fatalf("legacy route lost Engine compatibility:\n%s", source)
	}
	if strings.Contains(source, "Register(router gin.IRouter)") {
		t.Fatalf("legacy route unexpectedly uses router injection:\n%s", source)
	}
}

func TestBizContractTemplateUsesTransportNeutralTypes(t *testing.T) {
	g := &Generator{PackageName: "biz", Name: "user"}
	parsed, err := template.New("biz_contract").Parse(tpl.BizContractTpl)
	if err != nil {
		t.Fatal(err)
	}

	var rendered strings.Builder
	if err := parsed.Execute(&rendered, g); err != nil {
		t.Fatal(err)
	}

	source := rendered.String()
	for _, want := range []string{"type UserListQuery struct", "type CreateUserCommand struct", "type UpdateUserCommand struct"} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated contract is missing %q:\n%s", want, source)
		}
	}
}
