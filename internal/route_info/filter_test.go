package route_info

import "testing"

func TestApplyFilterByPackage(t *testing.T) {
	pr := &ProjectRoutes{
		Module: "test",
		Routes: []RouteInfo{
			{Handler: "list", FullPath: "/v1/notice", File: "internal/routes/v1/notice_routes.go"},
			{Handler: "doc", FullPath: "/open/disk/doc", File: "internal/routes/open/disk_routes.go"},
			{Handler: "search", FullPath: "/app/search/disk", File: "internal/routes/app/search_routes.go"},
		},
	}

	got := ApplyFilter(pr, &FilterOption{Package: "v1"})
	if len(got.Routes) != 1 || got.Routes[0].FullPath != "/v1/notice" {
		t.Fatalf("unexpected v1 filter: %#v", got.Routes)
	}

	got = ApplyFilter(pr, &FilterOption{Package: "open"})
	if len(got.Routes) != 1 || got.Routes[0].FullPath != "/open/disk/doc" {
		t.Fatalf("unexpected open filter: %#v", got.Routes)
	}

	got = ApplyFilter(pr, &FilterOption{Package: "app"})
	if len(got.Routes) != 1 || got.Routes[0].FullPath != "/app/search/disk" {
		t.Fatalf("unexpected app filter: %#v", got.Routes)
	}

	// Package + keyword combined.
	got = ApplyFilter(pr, &FilterOption{Package: "open", Keyword: "doc"})
	if len(got.Routes) != 1 || got.Routes[0].Handler != "doc" {
		t.Fatalf("unexpected combined filter: %#v", got.Routes)
	}

	// Keyword from another package must not leak in.
	got = ApplyFilter(pr, &FilterOption{Package: "v1", Keyword: "search"})
	if len(got.Routes) != 0 {
		t.Fatalf("expected no match, got %#v", got.Routes)
	}
}

func TestApplyFilterPackageTrailingSlash(t *testing.T) {
	pr := &ProjectRoutes{
		Routes: []RouteInfo{
			{Handler: "a", File: "internal/routes/v1/a_routes.go"},
			{Handler: "b", File: "internal/routes/open/b_routes.go"},
		},
	}

	got := ApplyFilter(pr, &FilterOption{Package: "v1/"})
	if len(got.Routes) != 1 || got.Routes[0].Handler != "a" {
		t.Fatalf("trailing slash not trimmed: %#v", got.Routes)
	}
}
