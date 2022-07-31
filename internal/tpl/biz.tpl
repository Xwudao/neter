{{- /*gotype: github.com/Xwudao/neter/cmd/neter/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

type {{.StructBizName}} struct {
}

func New{{.StructBizName}}() *{{.StructBizName}} {
	return &{{.StructBizName}}{}
}

type {{.ToCamel .Name}}Repository interface {
	TodoFunc() error
}

func (h *{{.StructBizName}}) Index() string {
	panic("TODO implement")
}
