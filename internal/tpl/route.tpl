{{- /*gotype: github.com/Xwudao/neter/cmd/neter/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"
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
	r.g.GET("/{{.PackageName}}/hello", r.{{.Name}}())
}

func (r *{{.StructName}}) {{.Name}}() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")

		c.String(200, "hello"+name)
	}
}
