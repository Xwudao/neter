{{- /*gotype: github.com/Xwudao/neter/cmd/neter/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"


	"{{.ModName}}/internal/core"
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
	r.g.GET("/{{.PackageName}}/{{.ToLowerCamel .Name}}", core.WrapData(r.{{.ToLowerCamel .Name}}()))
}


func (r *{{.StructRouteName}}) {{.ToLowerCamel .Name}}() core.WrappedHandlerFunc {
	return func(c *gin.Context) (interface{}, *core.RtnStatus) {
		return "hello", nil
	}
}
