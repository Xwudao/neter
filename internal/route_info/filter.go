package route_info

import "strings"

// FilterOption controls how routes are filtered.
type FilterOption struct {
	// Keyword filters by handler name or full path (case-insensitive, substring match).
	Keyword string
	// Package filters by route package directory (e.g. v1, app, open).
	// Matches handler files under internal/routes/<pkg>/.
	Package string
}

// ApplyFilter returns a new ProjectRoutes with only matching routes.
// If filter is nil or empty, returns a copy of all routes.
func ApplyFilter(pr *ProjectRoutes, filter *FilterOption) *ProjectRoutes {
	out := &ProjectRoutes{
		Module: pr.Module,
	}

	if filter == nil || (filter.Keyword == "" && filter.Package == "") {
		out.Routes = pr.Routes
		return out
	}

	keyword := strings.ToLower(filter.Keyword)
	pkgMarker := ""
	if filter.Package != "" {
		pkgMarker = "/routes/" + strings.Trim(filter.Package, "/") + "/"
	}

	for _, r := range pr.Routes {
		if keyword != "" &&
			!strings.Contains(strings.ToLower(r.Handler), keyword) &&
			!strings.Contains(strings.ToLower(r.FullPath), keyword) {
			continue
		}
		if pkgMarker != "" && !strings.Contains(r.File, pkgMarker) {
			continue
		}
		out.Routes = append(out.Routes, r)
	}

	return out
}
