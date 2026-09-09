{{- /*gotype: github.com/Xwudao/neter/internal/gen.Generator*/ -}}
package data

import (
	"context"
	"errors"

	"{{.ModName}}/internal/biz"
	"{{.ModName}}/internal/data/sqlc"
	"{{.ModName}}/internal/system"
)

var _ biz.{{.ToCamel .Name}}Repository = (*{{.ToLowerCamel .Name}}Repository)(nil)

type {{.ToLowerCamel .Name}}Repository struct {
	appCtx *system.AppContext
	data   *Data
}

func New{{.ToCamel .Name}}Repository(appCtx *system.AppContext, data *Data) biz.{{.ToCamel .Name}}Repository {
	return &{{.ToLowerCamel .Name}}Repository{appCtx: appCtx, data: data}
}

// err{{.Model}}NotImplemented is returned by the generated scaffolding. Add the
// CRUD queries to db/query/{{.Table}}.sql using the conventional names:
//
//	-- name: List{{.Plural}} :many
//	-- name: Get{{.Model}} :one
//	-- name: Create{{.Model}} :one
//	-- name: Update{{.Model}} :one
//	-- name: Delete{{.Model}} :execrows
//
// then run `make sqlc` and replace each body with a data.Queries call, e.g.
//
//	return r.data.Queries.List{{.Plural}}(ctx)
var err{{.Model}}NotImplemented = errors.New("{{.ToCamel .Name}}Repository: add CRUD queries to db/query/{{.Table}}.sql and run make sqlc")

func (r *{{.ToLowerCamel .Name}}Repository) GetAll(ctx context.Context) ([]*sqlc.{{.Model}}, error) {
	return nil, err{{.Model}}NotImplemented
}

func (r *{{.ToLowerCamel .Name}}Repository) DeleteByID(ctx context.Context, id int64) error {
	return err{{.Model}}NotImplemented
}

func (r *{{.ToLowerCamel .Name}}Repository) GetByID(ctx context.Context, id int64) (*sqlc.{{.Model}}, error) {
	return nil, err{{.Model}}NotImplemented
}

func (r *{{.ToLowerCamel .Name}}Repository) Create(ctx context.Context) (*sqlc.{{.Model}}, error) {
	return nil, err{{.Model}}NotImplemented
}
