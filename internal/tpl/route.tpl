{{- /*gotype: github.com/Xwudao/neter/cmd/neter/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"


	"{{.ModName}}/internal/core"
	"{{.ModName}}/internal/routes/mdw"
)

type {{.StructRouteName}} struct {
	conf *koanf.Koanf
	g    *gin.Engine
}

func New{{.StructRouteName}}(g *gin.Engine, conf *koanf.Koanf) *{{.StructRouteName}} {
	r := &{{.StructRouteName}}{
		conf: conf,
		g:    g,
	}

	return r
}

func (r *{{.StructRouteName}}) Reg() {
	// r.g.GET("/{{.PackageName}}/{{.ToLowerCamel .Name}}", core.WrapData(r.{{.ToLowerCamel .Name}}()))

	group := r.g.Group("/{{.PackageName}}/{{.ToLowerCamel .Name}}")
	{
		group.GET("", core.WrapData(r.{{.ToLowerCamel .Name}}()))
	}
	authGroup := r.g.Group("/auth/{{.PackageName}}/{{.ToLowerCamel .Name}}").Use(mdw.MustLoginMiddleware())
	{
		// authGroup.GET("/auth", core.WrapData(r.{{.ToLowerCamel .Name}}()))
		_ = authGroup
	}
	adminGroup := r.g.Group("/admin/{{.PackageName}}/{{.ToLowerCamel .Name}}").Use(mdw.MustWithRoleMiddleware("admin"))
	{
		_ = adminGroup
	}
}


func (r *{{.StructRouteName}}) {{.ToLowerCamel .Name}}() core.WrappedHandlerFunc {
	return func(c *gin.Context) (interface{}, *core.RtnStatus) {
		return "hello", nil
	}
}
