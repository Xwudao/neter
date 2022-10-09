{{- /*gotype: github.com/Xwudao/nr/cmd/nr/cmd.GenerateRoute*/ -}}
package {{.PackageName}}

type {{.ToCamel .Name}}Repository interface {
	TodoFunc() error
}

type {{.StructBizName}} struct {
}

func New{{.StructBizName}}() *{{.StructBizName}} {
	return &{{.StructBizName}}{}
}

func (h *{{.StructBizName}}) Index() string {
	panic("TODO implement")
}
