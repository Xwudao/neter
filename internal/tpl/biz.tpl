{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"context"

	"go.uber.org/zap"

	"{{.ModName}}/internal/system"{{if .WithCRUD}}
	"{{.ModName}}/internal/data/ent"
{{end}}
)

type {{.ToCamel .Name}}Repository interface {
{{if .WithCRUD}}
	GetAll() ([]*ent.{{.EntName}}, error)
	DeleteByID(id int64) error
	GetByID(id int64) (*ent.{{.EntName}}, error)
	Create() (*ent.{{.EntName}}, error)
{{else}}
	TodoFunc() error
{{end}}
}

type {{.StructBizName}} struct {
	log *zap.SugaredLogger
	ctx context.Context
	{{.ExtractInitials .Name}}r {{.ToCamel .Name}}Repository
}

func New{{.StructBizName}}(log *zap.SugaredLogger, {{.ExtractInitials .Name}}r {{.ToCamel .Name}}Repository, appCtx *system.AppContext) *{{.StructBizName}} {
	return &{{.StructBizName}}{
		log: log.Named("{{.ToKebab .StructBizName}}"),
		ctx: appCtx.Ctx,
		{{.ExtractInitials .Name}}r: {{.ExtractInitials .Name}}r,
	}
}

func (h *{{.StructBizName}}) Index() string {
	panic("TODO implement")
}

{{if .WithCRUD}}
	func (h *{{.StructBizName}}) Delete(id int64) error {
	return h.mr.DeleteByID(id)
	}

	func (h *{{.StructBizName}}) Get(id int64) (*ent.{{.EntName}}, error) {
	return h.mr.GetByID(id)
	}

	func (h *{{.StructBizName}}) Create() (*ent.{{.EntName}}, error) {
	return h.mr.Create()
	}

	func (h *{{.StructBizName}}) GetAll() ([]*ent.{{.EntName}}, error) {
	return h.mr.GetAll()
	}
{{end}}