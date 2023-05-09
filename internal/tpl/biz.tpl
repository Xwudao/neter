{{- /*gotype: github.com/Xwudao/nr/cmd/nr/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"go.uber.org/zap"
)

type {{.ToCamel .Name}}Repository interface {
	TodoFunc() error
}

type {{.StructBizName}} struct {
	log *zap.SugaredLogger
}

func New{{.StructBizName}}(log *zap.SugaredLogger) *{{.StructBizName}} {
	return &{{.StructBizName}}{
		log: log.Named("{{.ToKebab .StructBizName}}"),
	}
}

func (h *{{.StructBizName}}) Index() string {
	panic("TODO implement")
}
