{{- /*gotype: github.com/Xwudao/neter/cmd/neter/cmd.GenerateRoute*/ -}}
package data

import (
	"{{.ModName}}/internal/biz"
)

var _ biz.{{.ToCamel .Name}}Repository = (*{{.ToLowerCamel .Name}}Repository)(nil)

type {{.ToLowerCamel .Name}}Repository struct {
}

func (u *{{.ToLowerCamel .Name}}Repository) TodoFunc() error {
	//TODO implement me
	panic("implement me")
}
func New{{.ToCamel .Name}}Repository() biz.{{.ToCamel .Name}}Repository {
	return &{{.ToLowerCamel .Name}}Repository{}
}
