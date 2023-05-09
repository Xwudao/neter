{{- /*gotype: github.com/Xwudao/nr/cmd/nr/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"context"

	"go.uber.org/zap"

	"{{.ModName}}/internal/system"
)

type {{.ToCamel .Name}}Repository interface {
	TodoFunc() error
}

type {{.StructBizName}} struct {
	log *zap.SugaredLogger
	ctx context.Context
}

func New{{.StructBizName}}(log *zap.SugaredLogger, appCtx *system.AppContext) *{{.StructBizName}} {
	return &{{.StructBizName}}{
		log: log.Named("{{.ToKebab .StructBizName}}"),
		ctx: appCtx.Ctx,
	}
}

func (h *{{.StructBizName}}) Index() string {
	panic("TODO implement")
}
