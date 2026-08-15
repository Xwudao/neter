package route_info

import (
	"strings"
	"testing"
)

func TestGenerateTypeScript(t *testing.T) {
	routes := &ProjectRoutes{Routes: []RouteInfo{
		{
			Method:   "POST",
			FullPath: "/v1/user/login",
			Params: []ParamInfo{{
				Source: "body",
				Fields: []FieldInfo{
					{Name: "Username", Type: "string", Tag: `json:"username" binding:"required"`},
					{Name: "Remember", Type: "bool", Tag: `json:"remember"`},
				},
			}},
			Returns: []ReturnInfo{{
				Type: "LoginResponse",
				Fields: []FieldInfo{
					{Name: "Token", Type: "string", Tag: `json:"token"`},
					{Name: "Profile", Type: "*Profile", Tag: `json:"profile,omitempty"`, Fields: []FieldInfo{{Name: "ID", Type: "int64", Tag: `json:"id"`}}},
				},
			}},
		},
		{
			Method:   "GET",
			FullPath: "/admin/v1/items",
			Params: []ParamInfo{{
				Source: "request",
				Fields: []FieldInfo{{Name: "Page", Type: "int", Tag: `form:"page"`}},
			}},
			Returns: []ReturnInfo{{Type: "[]string"}},
		},
		{
			Method:   "GET",
			FullPath: "/v1/users/:id",
			Returns: []ReturnInfo{{
				Type: "ent.User",
				Fields: []FieldInfo{
					{Name: "ID", Type: "int", Tag: `json:"id,omitempty"`, FromEnt: true},
					{Name: "Nickname", Type: "string", Tag: `json:"nickname,omitempty"`, FromEnt: true},
					{Name: "Bio", Type: "*string", Tag: `json:"bio,omitempty"`, FromEnt: true},
					{Name: "Cursor", Type: "string", Tag: `json:"cursor,omitempty"`},
				},
			}},
		},
	}}

	got := GenerateTypeScript(routes)
	for _, want := range []string{
		"export interface PostV1UserLoginBody",
		"username: string",
		"remember?: boolean",
		"export interface PostV1UserLoginResponseData",
		"export type PostV1UserLoginResponse = ApiResponse<PostV1UserLoginResponseData>",
		"export interface GetAdminV1ItemsQuery",
		"page?: number",
		"export type GetAdminV1ItemsResponse = ApiResponse<Array<string>>",
		"id: number",
		"nickname: string",
		"bio?: string | null",
		"cursor?: string",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated TypeScript missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"id?: number", "nickname?: string"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("generated TypeScript unexpectedly made Ent field optional %q:\n%s", unwanted, got)
		}
	}
}

func TestTsGenFileNames(t *testing.T) {
	srcFiles := []string{
		"internal/routes/v1/category_routes.go", // unique base → bare name
		"internal/routes/v1/search_routes.go",   // collides: primary (largest path) keeps bare name
		"internal/routes/app/search_routes.go",  // collides: parent-dir prefix
		"internal/routes/v1/disk_routes.go",     // collides: primary keeps bare name
		"internal/routes/app/disk_routes.go",    // collides: parent-dir prefix
		"internal/routes/open/disk_routes.go",   // collides: parent-dir prefix
		"internal/routes/routes.go",             // base "api" → bare name
	}

	names := tsGenFileNames(srcFiles)
	got := map[string]string{}
	for _, src := range srcFiles {
		got[src] = names[src]
	}

	want := map[string]string{
		"internal/routes/v1/category_routes.go": "category.gen.ts",
		"internal/routes/v1/search_routes.go":   "search.gen.ts",
		"internal/routes/app/search_routes.go":  "app_search.gen.ts",
		"internal/routes/v1/disk_routes.go":     "disk.gen.ts",
		"internal/routes/app/disk_routes.go":    "app_disk.gen.ts",
		"internal/routes/open/disk_routes.go":   "open_disk.gen.ts",
		"internal/routes/routes.go":             "api.gen.ts",
	}
	for src, w := range want {
		if got[src] != w {
			t.Errorf("tsGenFileNames(%q) = %q, want %q", src, got[src], w)
		}
	}

	// Every name must be unique and complete.
	seen := map[string]string{}
	for src, name := range got {
		if prev, ok := seen[name]; ok {
			t.Errorf("output name %q assigned to both %q and %q", name, prev, src)
		}
		seen[name] = src
	}
	if len(seen) != len(srcFiles) {
		t.Errorf("expected %d distinct output names, got %d", len(srcFiles), len(seen))
	}
}

func TestTsGenFileNamesUniquenessFallback(t *testing.T) {
	// Pathological layout: a prefixed collision name collides with another
	// file's bare name. The fallback suffix must keep everything unique.
	srcFiles := []string{
		"internal/routes/v1/search_routes.go",
		"internal/routes/app/search_routes.go",
		"internal/routes/app_search.go", // bare name app_search.gen.ts
	}

	names := tsGenFileNames(srcFiles)
	seen := map[string]string{}
	for _, src := range srcFiles {
		name := names[src]
		if name == "" {
			t.Errorf("missing output name for %q", src)
			continue
		}
		if prev, ok := seen[name]; ok {
			t.Errorf("output name %q assigned to both %q and %q", name, prev, src)
		}
		seen[name] = src
	}
}
