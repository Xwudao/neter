{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.GenSubCmd*/ -}}
package cmd_app

import (
)

type {{.StructAppName}} struct {
}

func New{{.StructAppName}}() *{{.StructAppName}} {
	return &{{.StructAppName}}{
	}
}

func (a *{{.StructAppName}}) Run() {

}