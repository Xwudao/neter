{{- /*gotype: github.com/Xwudao/neter/cmd/neter/cmd.GenerateRoute*/ -}}
package {{.PackageName}}
import (
	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"
)

type {{.StructName}} struct {
	router *gin.Engine
	conf   *koanf.Koanf
}

func New{{.StructName}}(router *gin.Engine, conf *koanf.Koanf) *{{.StructName}} {
	r := &{{.StructName}}{router: router, conf: conf}

	r.router.GET("/", r.{{.Name}}())

	return r
}

func (r *{{.StructName}}) {{.Name}}() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.String(200, "Hello World!")
	}
}
