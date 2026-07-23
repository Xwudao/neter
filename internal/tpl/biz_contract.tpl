{{- /*gotype: github.com/Xwudao/neter/internal/gen.Generator*/ -}}
package {{.PackageName}}

// {{.ToCamel .Name}}ListQuery contains pagination and filter inputs for a
// business read. Keep HTTP binding tags in internal/domain/params; routes map
// those transport inputs to this type before calling the biz layer.
type {{.ToCamel .Name}}ListQuery struct {
	Offset int
	Limit  int
}

// Create{{.ToCamel .Name}}Command contains the data required to create a
// {{.ToCamel .Name}}. Add domain fields here instead of passing a Gin request
// type into the biz layer.
type Create{{.ToCamel .Name}}Command struct {
}

// Update{{.ToCamel .Name}}Command contains the data required to update a
// {{.ToCamel .Name}}. Add domain fields here as the use case grows.
type Update{{.ToCamel .Name}}Command struct {
	ID int64
}
