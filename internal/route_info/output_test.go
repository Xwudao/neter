package route_info

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildBodyJSONSchemaProducesValidNestedJSON(t *testing.T) {
	body := buildBodyJSONSchema([]ParamInfo{{
		Fields: []FieldInfo{
			{Name: "Name", Type: "string", Tag: `json:"name"`},
			{Name: "Profile", Type: "Profile", Tag: `json:"profile"`, Fields: []FieldInfo{
				{Name: "Email", Type: "string", Tag: `json:"email"`},
			}},
			{Name: "Tags", Type: "[]Tag", Tag: `json:"tags"`, Fields: []FieldInfo{
				{Name: "Value", Type: "string", Tag: `json:"value"`},
			}},
		},
	}})

	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("generated body is not JSON: %v\n%s", err, body)
	}
	profile, ok := got["profile"].(map[string]any)
	if !ok || profile["email"] != "email" {
		t.Fatalf("profile = %#v, want nested email example", got["profile"])
	}
	tags, ok := got["tags"].([]any)
	if !ok || len(tags) != 1 {
		t.Fatalf("tags = %#v, want one nested array item", got["tags"])
	}
}

func TestGenerateTerminalRouteUsesQueryFormTagsInCurlAndTable(t *testing.T) {
	route := RouteInfo{
		Method:   "GET",
		FullPath: "/admin/v1/disk/list",
		Params: []ParamInfo{{
			Source:     "query",
			StructType: "params.ListDiskParams",
			Fields: []FieldInfo{
				{Name: "Page", Type: "int", Tag: `json:"page" form:"page" binding:"required"`, Required: true},
				{Name: "WithCate", Type: "*bool", Tag: `json:"with_cate" form:"with_cate"`},
			},
		}},
	}

	got := generateTerminalRoute(route, "http://localhost:4677")
	if !strings.Contains(got, "curl 'http://localhost:4677/admin/v1/disk/list?page={page}&with_cate={with_cate}'") {
		t.Fatalf("curl output = %q, want query parameters from form tags", got)
	}
	if !strings.Contains(got, "| page | int | ✓ |") || !strings.Contains(got, "| with_cate | *bool |  |") {
		t.Fatalf("query table = %q, want form tag names", got)
	}
}
