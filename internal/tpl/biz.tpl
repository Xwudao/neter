{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.Generator*/ -}}
package {{.PackageName}}

import (
	{{if .WithCRUD}}
		"context"
{{end}}

	"go.uber.org/zap"

	"{{.ModName}}/internal/system"{{if .WithCRUD}}
	"{{.ModName}}/internal/data/ent"
{{end}}
)

type {{.ToCamel .Name}}Repository interface {
{{if .WithCRUD}}
	GetAll(ctx context.Context) ([]*ent.{{.EntName}}, error)
	DeleteByID(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*ent.{{.EntName}}, error)
	Create(ctx context.Context) (*ent.{{.EntName}}, error)
{{else}}
	TodoFunc() error
{{end}}
}

type {{.StructBizName}} struct {
	log *zap.SugaredLogger
	appCtx *system.AppContext
	{{.ExtractInitials .Name}}r {{.ToCamel .Name}}Repository
}

func New{{.StructBizName}}(log *zap.SugaredLogger, {{.ExtractInitials .Name}}r {{.ToCamel .Name}}Repository, appCtx *system.AppContext) *{{.StructBizName}} {
	return &{{.StructBizName}}{
		log: log.Named("{{.ToKebab .StructBizName}}"),
		appCtx: appCtx,
		{{.ExtractInitials .Name}}r: {{.ExtractInitials .Name}}r,
	}
}

func (h *{{.StructBizName}}) Index() string {
	panic("TODO implement")
}

{{if .WithCRUD}}
	func (h *{{.StructBizName}}) Delete(ctx context.Context, id int64) error {
	return h.{{.ExtractInitials .Name}}r.DeleteByID(ctx, id)
	}

	func (h *{{.StructBizName}}) Get(ctx context.Context, id int64) (*ent.{{.EntName}}, error) {
	return h.{{.ExtractInitials .Name}}r.GetByID(ctx, id)
	}

	func (h *{{.StructBizName}}) Create(ctx context.Context) (*ent.{{.EntName}}, error) {
	return h.{{.ExtractInitials .Name}}r.Create(ctx)
	}

	func (h *{{.StructBizName}}) GetAll(ctx context.Context) ([]*ent.{{.EntName}}, error) {
	return h.{{.ExtractInitials .Name}}r.GetAll(ctx)
	}
{{end}}