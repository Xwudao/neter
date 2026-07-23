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
	log        *zap.SugaredLogger
}

func New{{.StructRouteName}}(log *zap.SugaredLogger, conf *koanf.Koanf) *{{.StructRouteName}} {
	r := &{{.StructRouteName}}{
		conf: conf,
		log:  log.Named("{{.ToKebab .StructRouteName}}"),
	}

	return r
}

func (r *{{.StructRouteName}}) {{if .UseRouteRegistry}}Register{{else}}Reg{{end}}(router gin.IRouter) {
	// router.GET("/{{.PackageName}}/{{.ToSnake .Name}}", {{if .UseTypedAPI}}core.NoInput(r.{{.ToLowerCamel .Name}}){{else}}core.WrapData(r.{{.ToLowerCamel .Name}}()){{end}})

	group := router.Group("/{{.PackageName}}/{{.ToSnake .Name}}")
	{
		group.GET("", {{if .UseTypedAPI}}core.NoInput(r.{{.ToLowerCamel .Name}}){{else}}core.WrapData(r.{{.ToLowerCamel .Name}}()){{end}})
	}
	authGroup := router.Group("/auth/{{.PackageName}}/{{.ToSnake .Name}}").Use(mdw.MustLoginMiddleware())
	{
		// authGroup.GET("/auth", {{if .UseTypedAPI}}core.NoInput(r.{{.ToLowerCamel .Name}}){{else}}core.WrapData(r.{{.ToLowerCamel .Name}}()){{end}})
		_ = authGroup
	}
	adminGroup := router.Group("/admin/{{.PackageName}}/{{.ToSnake .Name}}").Use(mdw.MustWithRoleMiddleware(user.RoleAdmin))
	{
		_ = adminGroup
	}
}


{{if .UseTypedAPI -}}
func (r *{{.StructRouteName}}) {{.ToLowerCamel .Name}}(c *gin.Context) (string, *core.RtnStatus) {
	return "hello", nil
}
{{else -}}
func (r *{{.StructRouteName}}) {{.ToLowerCamel .Name}}() core.WrappedHandlerFunc {
	return func(c *gin.Context) (any, *core.RtnStatus) {
		return "hello", nil
	}
}
{{end}}
