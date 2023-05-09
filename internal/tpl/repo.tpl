{{- /*gotype: github.com/Xwudao/nr/cmd/nr/cmd.GenerateRoute*/ -}}
package data

import (
	"context"

	"{{.ModName}}/internal/biz"
	"{{.ModName}}/internal/system"
)

var _ biz.{{.ToCamel .Name}}Repository = (*{{.ToLowerCamel .Name}}Repository)(nil)

type {{.ToLowerCamel .Name}}Repository struct {
	ctx context.Context
}

func (u *{{.ToLowerCamel .Name}}Repository) TodoFunc() error {
	//TODO implement me
	panic("implement me")
}
func New{{.ToCamel .Name}}Repository(appCtx *system.AppContext) biz.{{.ToCamel .Name}}Repository {
	return &{{.ToLowerCamel .Name}}Repository{
		ctx: appCtx.Ctx,
	}
}
