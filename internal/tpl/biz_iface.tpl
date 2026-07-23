{{- /*gotype: github.com/Xwudao/neter/internal/gen.Generator*/ -}}
package {{.PackageName}}

{{if .WithCRUD}}
import (
	"context"

	"{{.ModName}}/internal/data/ent"
)
{{end}}

// {{.StructBizName}}Iface is the interface consumed by route handlers.
// It is defined alongside its implementation so the biz package remains the single
// source of truth. Use wire.Bind(new({{.StructBizName}}Iface), new(*{{.StructBizName}})) in
// the biz provider set so Wire can inject this interface into route constructors.
type {{.StructBizName}}Iface interface {
	{{if .WithCRUD}}
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*ent.{{.EntName}}, error)
	Create(ctx context.Context) (*ent.{{.EntName}}, error)
	GetAll(ctx context.Context) ([]*ent.{{.EntName}}, error)
	{{else}}
	// TODO: add methods that route handlers need.
	{{end}}
}

// Compile-time assertion: *{{.StructBizName}} must satisfy {{.StructBizName}}Iface.
var _ {{.StructBizName}}Iface = (*{{.StructBizName}})(nil)
