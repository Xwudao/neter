{{- /*gotype: github.com/Xwudao/neter/cmd/neter/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"


	"{{.ModName}}/internal/core"
)

type {{.StructName}} struct {
	conf *koanf.Koanf
	g    *gin.Engine
}

func New{{.StructName}}(g *gin.Engine, conf *koanf.Koanf) *{{.StructName}} {
	r := &{{.StructName}}{
		conf: conf,
		g:    g,
	}

	return r
}

func (r *{{.StructName}}) Reg() {
	r.g.GET("/{{.PackageName}}/{{.ToLowerCamel .Name}}", core.WrapData(r.{{.ToLowerCamel .Name}}()))
}


func (r *{{.StructName}}) {{.ToLowerCamel .Name}}() core.WrappedHandlerFunc {
	return func(c *gin.Context) (interface{}, *core.RtnStatus) {
		return "hello", nil
	}
}
