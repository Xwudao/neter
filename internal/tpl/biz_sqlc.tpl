{{- /*gotype: github.com/Xwudao/neter/internal/gen.Generator*/ -}}
package {{.PackageName}}

import (
	"context"

	"go.uber.org/zap"

	"{{.ModName}}/internal/data/sqlc"
	"{{.ModName}}/internal/system"
)

type {{.ToCamel .Name}}Repository interface {
	GetAll(ctx context.Context) ([]*sqlc.{{.Model}}, error)
	DeleteByID(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*sqlc.{{.Model}}, error)
	Create(ctx context.Context) (*sqlc.{{.Model}}, error)
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

func (h *{{.StructBizName}}) Delete(ctx context.Context, id int64) error {
	return h.{{.ExtractInitials .Name}}r.DeleteByID(ctx, id)
}

func (h *{{.StructBizName}}) Get(ctx context.Context, id int64) (*sqlc.{{.Model}}, error) {
	return h.{{.ExtractInitials .Name}}r.GetByID(ctx, id)
}

func (h *{{.StructBizName}}) Create(ctx context.Context) (*sqlc.{{.Model}}, error) {
	return h.{{.ExtractInitials .Name}}r.Create(ctx)
}

func (h *{{.StructBizName}}) GetAll(ctx context.Context) ([]*sqlc.{{.Model}}, error) {
	return h.{{.ExtractInitials .Name}}r.GetAll(ctx)
}
