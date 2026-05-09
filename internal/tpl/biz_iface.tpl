{{- /*gotype: github.com/Xwudao/neter/internal/gen.Generator*/ -}}
package {{.PackageName}}

// {{.StructBizName}}Iface is the interface consumed by route handlers.
// It is defined alongside its implementation so the biz package remains the single
// source of truth. Use wire.Bind(new({{.StructBizName}}Iface), new(*{{.StructBizName}})) in
// the biz provider set so Wire can inject this interface into route constructors.
type {{.StructBizName}}Iface interface {
	// TODO: add methods that route handlers need.
}

// Compile-time assertion: *{{.StructBizName}} must satisfy {{.StructBizName}}Iface.
var _ {{.StructBizName}}Iface = (*{{.StructBizName}})(nil)
