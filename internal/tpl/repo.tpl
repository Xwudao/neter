{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.GenerateRoute*/ -}}
package data

import (
	"context"

	"{{.ModName}}/internal/biz"
	"{{.ModName}}/internal/system"
{{- if .WithCRUD}}
		"{{.ModName}}/internal/data/ent"
{{end}}
)

var _ biz.{{.ToCamel .Name}}Repository = (*{{.ToLowerCamel .Name}}Repository)(nil)

type {{.ToLowerCamel .Name}}Repository struct {
	ctx context.Context
{{- if .WithCRUD}}
	data *Data
{{- end}}
}

func New{{.ToCamel .Name}}Repository(appCtx *system.AppContext {{if .WithCRUD}}, data *Data{{end}}) biz.{{.ToCamel .Name}}Repository {
	return &{{.ToLowerCamel .Name}}Repository{
		ctx: appCtx.Ctx,
{{- if .WithCRUD}}
		data: data,
{{- end}}
	}
}

{{if .WithCRUD}}
	func (u *{{.ToLowerCamel .Name}}Repository) GetAll() ([]*ent.{{.EntName}}, error) {
	return u.data.Client.{{.EntName}}.Query().All(u.ctx)
	}

	func (u *{{.ToLowerCamel .Name}}Repository) DeleteByID(id int64) error {
	return u.data.Client.{{.EntName}}.DeleteOneID(id).Exec(u.ctx)
	}

	func (u *{{.ToLowerCamel .Name}}Repository) GetByID(id int64) (*ent.{{.EntName}}, error) {
	return u.data.Client.{{.EntName}}.Get(u.ctx, id)
	}

	func (u *{{.ToLowerCamel .Name}}Repository) Create() (*ent.{{.EntName}}, error) {
	// todo add set fields
	return u.data.Client.{{.EntName}}.Create().Save(u.ctx)
	}
{{else}}
    func (u *{{.ToLowerCamel .Name}}Repository) TodoFunc() error {
    //TODO implement me
    panic("implement me")
    }
{{end}}