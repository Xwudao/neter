package visitor

import (
	"go/ast"
	"go/token"
)

type CurrentProvideVisitor struct {
	FunName string
}

func NewCurrentProvideVisitor(funName string) *CurrentProvideVisitor {
	return &CurrentProvideVisitor{FunName: funName}
}

func (v *CurrentProvideVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Tok == token.VAR && len(n.Specs) > 0 {
			specs := n.Specs[0].(*ast.ValueSpec)
			if len(specs.Values) > 0 {
				val := specs.Values[0].(*ast.CallExpr)
				val.Args = append(val.Args,
					ast.NewIdent(v.FunName),
				)
			}
		}
	}
	return v
}
