{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.GenerateRoute*/ -}}
package data

import (
{{if .WithCRUD}}
	"context"
{{end}}

	"{{.ModName}}/internal/biz"
	"{{.ModName}}/internal/system"
{{- if .WithCRUD}}
		"{{.ModName}}/internal/data/ent"
{{end}}
)

var _ biz.{{.ToCamel .Name}}Repository = (*{{.ToLowerCamel .Name}}Repository)(nil)

type {{.ToLowerCamel .Name}}Repository struct {
	appCtx *system.AppContext
{{- if .WithCRUD}}
	data *Data
{{- end}}
}

func New{{.ToCamel .Name}}Repository(appCtx *system.AppContext {{if .WithCRUD}}, data *Data{{end}}) biz.{{.ToCamel .Name}}Repository {
	return &{{.ToLowerCamel .Name}}Repository{
		appCtx: appCtx,
{{- if .WithCRUD}}
		data: data,
{{- end}}
	}
}

{{if .WithCRUD}}
	func (u *{{.ToLowerCamel .Name}}Repository) GetAll(ctx context.Context) ([]*ent.{{.EntName}}, error) {
	return u.data.Client.{{.EntName}}.Query().All(ctx)
	}

	func (u *{{.ToLowerCamel .Name}}Repository) DeleteByID(ctx context.Context,id int64) error {
	return u.data.Client.{{.EntName}}.DeleteOneID(id).Exec(ctx)
	}

	func (u *{{.ToLowerCamel .Name}}Repository) GetByID(ctx context.Context,id int64) (*ent.{{.EntName}}, error) {
	return u.data.Client.{{.EntName}}.Get(ctx, id)
	}

	func (u *{{.ToLowerCamel .Name}}Repository) Create(ctx context.Context) (*ent.{{.EntName}}, error) {
	// todo add set fields
	return u.data.Client.{{.EntName}}.Create().Save(ctx)
	}
{{else}}
    func (u *{{.ToLowerCamel .Name}}Repository) TodoFunc() error {
    //TODO implement me
    panic("implement me")
    }
{{end}}