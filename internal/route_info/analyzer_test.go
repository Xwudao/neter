package route_info

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeRoutesScansCustomLayoutAndExpandsNestedFields(t *testing.T) {
	root := t.TempDir()
	writeRouteInfoFixture(t, root, "go.mod", "module example.test/app\n\ngo 1.24\n")
	writeRouteInfoFixture(t, root, "api/user_route.go", `package api

import (
    "example.test/app/params"
    "example.test/core"
    "github.com/gin-gonic/gin"
)

type UserRoute struct { g *gin.RouterGroup }

func (r *UserRoute) Reg() {
    group := r.g.Group("/api")
    group.POST("/users", core.WrapData(r.create()))
}

func (r *UserRoute) create() core.WrappedHandlerFunc {
    return func(c *gin.Context) (any, *core.RtnStatus) {
        var payload params.CreateUserRequest
        _ = c.ShouldBindJSON(&payload)
        return payload, nil
    }
}
`)
	writeRouteInfoFixture(t, root, "params/request.go", `package params

type CreateUserRequest struct {
    Name string `+"`json:\"name\" binding:\"required,alphanum,min=6,max=20\"`"+`
    Profile Profile `+"`json:\"profile\"`"+`
    Tags []Tag `+"`json:\"tags\"`"+`
}

type Profile struct { Email string `+"`json:\"email\" binding:\"required\"`"+` }
type Tag struct { Value string `+"`json:\"value\"`"+` }
`)

	routes, err := AnalyzeRoutes(root)
	if err != nil {
		t.Fatalf("AnalyzeRoutes() error = %v", err)
	}
	if len(routes.Routes) != 1 {
		t.Fatalf("route count = %d, want 1", len(routes.Routes))
	}

	route := routes.Routes[0]
	if route.FullPath != "/api/users" || route.File != "api/user_route.go" {
		t.Fatalf("route = %#v, want custom-layout route", route)
	}
	if len(route.Params) != 1 || len(route.Params[0].Fields) != 3 {
		t.Fatalf("params = %#v, want expanded request fields", route.Params)
	}
	if !route.Params[0].Fields[0].Required {
		t.Fatalf("name field = %#v, want required with additional binding rules", route.Params[0].Fields[0])
	}
	if got := route.Params[0].Fields[1].Fields; len(got) != 1 || got[0].Name != "Email" {
		t.Fatalf("profile fields = %#v, want Email", got)
	}
	if got := route.Params[0].Fields[2].Fields; len(got) != 1 || got[0].Name != "Value" {
		t.Fatalf("tag fields = %#v, want Value", got)
	}
}

func TestAnalyzeRoutesFindsNestedGroupsMiddlewareAndAdditionalInputs(t *testing.T) {
	root := t.TempDir()
	writeRouteInfoFixture(t, root, "go.mod", "module example.test/app\n\ngo 1.24\n")
	writeRouteInfoFixture(t, root, "api/asset_route.go", `package api

import (
    "example.test/core"
    "github.com/gin-gonic/gin"
)

type AssetRoute struct { g *gin.Engine }

func (r *AssetRoute) Reg() {
    api := r.g.Group("/api")
    assets := api.Group("/assets").Use(auth())
    assets.POST("/:id", audit(), core.WrapData(r.upload()))
    assets.Any("/all", core.WrapData(r.list()))
    assets.Handle("PATCH", "/rename", core.WrapData(r.rename()))
}

func (r *AssetRoute) upload() core.WrappedHandlerFunc {
    return func(c *gin.Context) (any, *core.RtnStatus) {
        var input UploadInput
        _ = c.ShouldBindUri(&input)
        _ = c.GetHeader("Authorization")
        _ = c.PostForm("caption")
        _, _ = c.FormFile("file")
        return input, nil
    }
}

func (r *AssetRoute) list() core.WrappedHandlerFunc {
    return func(c *gin.Context) (any, *core.RtnStatus) {
        _ = c.QueryArray("tag")
        return nil, nil
    }
}

func (r *AssetRoute) rename() core.WrappedHandlerFunc {
    return func(c *gin.Context) (any, *core.RtnStatus) { return nil, nil }
}

func auth() gin.HandlerFunc { return nil }
func audit() gin.HandlerFunc { return nil }

type UploadInput struct { ID string `+"`uri:\"id\" binding:\"required\"`"+` }
`)

	routes, err := AnalyzeRoutes(root)
	if err != nil {
		t.Fatalf("AnalyzeRoutes() error = %v", err)
	}
	if len(routes.Routes) != 3 {
		t.Fatalf("route count = %d, want 3", len(routes.Routes))
	}

	upload := routes.Routes[0]
	if upload.FullPath != "/api/assets/:id" || upload.Line == 0 || len(upload.Middlewares) != 1 || upload.Middlewares[0] != "audit(...)" {
		t.Fatalf("upload route = %#v, want nested prefix and middleware", upload)
	}
	if !hasParam(upload.Params, "uri", "", "UploadInput") || !hasParam(upload.Params, "header", "Authorization", "") || !hasParam(upload.Params, "form", "caption", "") || !hasParam(upload.Params, "file", "file", "") {
		t.Fatalf("upload params = %#v, want uri, header, form, and file inputs", upload.Params)
	}
	if routes.Routes[1].Method != "ANY" || !hasParam(routes.Routes[1].Params, "query", "tag", "") {
		t.Fatalf("any route = %#v, want ANY and query array", routes.Routes[1])
	}
	if routes.Routes[2].Method != "PATCH" || routes.Routes[2].FullPath != "/api/assets/rename" {
		t.Fatalf("handle route = %#v, want PATCH /api/assets/rename", routes.Routes[2])
	}
}

func TestAnalyzeRoutesExpandsNestedMapResponseLiterals(t *testing.T) {
	root := t.TempDir()
	writeRouteInfoFixture(t, root, "go.mod", "module example.test/app\n\ngo 1.24\n")
	writeRouteInfoFixture(t, root, "api/status_route.go", `package api

import (
    "example.test/core"
    "github.com/gin-gonic/gin"
)

type StatusRoute struct{}

func (r *StatusRoute) Register(router gin.IRouter) {
    group := router.Group("/api")
    group.GET("/status", core.NoInput(r.status))
    group.GET("/items", core.NoInput(r.items))
}

func (r *StatusRoute) status(c *gin.Context) (map[string]any, *core.RtnStatus) {
    return gin.H{"name": "neter", "meta": gin.H{"enabled": true, "count": 2}, "items": []gin.H{{"id": 1}}}, nil
}

func (r *StatusRoute) items(c *gin.Context) ([]gin.H, *core.RtnStatus) {
    return []gin.H{{"id": 1, "name": "first"}}, nil
}
`)
	routes, err := AnalyzeRoutes(root)
	if err != nil {
		t.Fatal(err)
	}
	fields := routes.Routes[0].Returns[0].Fields
	if len(fields) != 3 || fields[1].Name != "meta" || len(fields[1].Fields) != 2 || fields[1].Fields[1].Type != "int" {
		t.Fatalf("nested fields = %#v", fields)
	}
	if fields[2].Name != "items" || len(fields[2].Fields) != 1 || fields[2].Fields[0].Name != "id" {
		t.Fatalf("nested array fields = %#v", fields[2])
	}
	itemFields := routes.Routes[1].Returns[0].Fields
	if len(itemFields) != 2 || itemFields[0].Name != "id" || itemFields[1].Name != "name" {
		t.Fatalf("array response fields = %#v", itemFields)
	}
}

func hasParam(params []ParamInfo, source, key, structType string) bool {
	for _, p := range params {
		if p.Source == source && p.Key == key && p.StructType == structType {
			return true
		}
	}
	return false
}

func writeRouteInfoFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
