package visitor

import (
	"go/ast"
	"go/token"
)

type ProvideVisitor struct {
	pkgName string
	FunName string
}

func NewProvideVisitor(pkgName string, funName string) *ProvideVisitor {
	return &ProvideVisitor{pkgName: pkgName, FunName: funName}
}

func (v *ProvideVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Tok == token.VAR && len(n.Specs) > 0 {
			specs := n.Specs[0].(*ast.ValueSpec)
			if len(specs.Values) > 0 {
				val := specs.Values[0].(*ast.CallExpr)
				val.Args = append(val.Args,
					&ast.SelectorExpr{
						X:   ast.NewIdent(v.pkgName),
						Sel: ast.NewIdent(v.FunName),
					},
				)
			}
		}
	}
	return v
}
