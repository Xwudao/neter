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
