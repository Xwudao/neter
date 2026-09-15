{{- /*gotype: github.com/Xwudao/neter/internal/gen.Generator*/ -}}
package {{.PackageName}}

import (
	"context"

	"{{.ModName}}/internal/data/sqlc"
)

// {{.StructBizName}}Iface is the interface consumed by route handlers.
// It is defined alongside its implementation so the biz package remains the single
// source of truth. Use loom.As[{{.StructBizName}}Iface](New{{.StructBizName}}) in the
// biz provider set so Loom can inject this interface into route constructors.
type {{.StructBizName}}Iface interface {
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*sqlc.{{.Model}}, error)
	Create(ctx context.Context) (*sqlc.{{.Model}}, error)
	GetAll(ctx context.Context) ([]*sqlc.{{.Model}}, error)
}

// Compile-time assertion: *{{.StructBizName}} must satisfy {{.StructBizName}}Iface.
var _ {{.StructBizName}}Iface = (*{{.StructBizName}})(nil)
