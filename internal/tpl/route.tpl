{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"
{{if .V2 -}}
	"github.com/knadh/koanf/v2"
{{else -}}
	"github.com/knadh/koanf"
{{- end}}
	"go.uber.org/zap"


	"{{.ModName}}/internal/core"
	"{{.ModName}}/internal/data/ent/user"
	"{{.ModName}}/internal/routes/mdw"
)

type {{.StructRouteName}} struct {
	conf *koanf.Koanf
	g    *gin.Engine
	log        *zap.SugaredLogger
}

func New{{.StructRouteName}}(g *gin.Engine, log *zap.SugaredLogger, conf *koanf.Koanf) *{{.StructRouteName}} {
	r := &{{.StructRouteName}}{
		conf: conf,
		g:    g,
		log:  log.Named("{{.ToKebab .StructRouteName}}"),
	}

	return r
}

func (r *{{.StructRouteName}}) {{if .UseRouteRegistry}}Register{{else}}Reg{{end}}() {
	// r.g.GET("/{{.PackageName}}/{{.ToSnake .Name}}", core.NoInput(r.{{.ToLowerCamel .Name}}))

	group := r.g.Group("/{{.PackageName}}/{{.ToSnake .Name}}")
	{
		group.GET("", core.NoInput(r.{{.ToLowerCamel .Name}}))
	}
	authGroup := r.g.Group("/auth/{{.PackageName}}/{{.ToSnake .Name}}").Use(mdw.MustLoginMiddleware())
	{
		// authGroup.GET("/auth", core.NoInput(r.{{.ToLowerCamel .Name}}))
		_ = authGroup
	}
	adminGroup := r.g.Group("/admin/{{.PackageName}}/{{.ToSnake .Name}}").Use(mdw.MustWithRoleMiddleware(user.RoleAdmin))
	{
		_ = adminGroup
	}
}


func (r *{{.StructRouteName}}) {{.ToLowerCamel .Name}}(c *gin.Context) (string, *core.RtnStatus) {
	return "hello", nil
}
