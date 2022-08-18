package visitor

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/go-toolsmith/strparse"
)

type RouteProvideVisitor struct {
	pkgName string
	FunName string
}

func NewRouteProvideVisitor(pkgName string, funName string) *RouteProvideVisitor {
	return &RouteProvideVisitor{pkgName: pkgName, FunName: funName}
}

func (v *RouteProvideVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Tok == token.VAR && len(n.Specs) > 0 {
			specs := n.Specs[0].(*ast.ValueSpec)
			if len(specs.Values) > 0 {
				val := specs.Values[0].(*ast.CallExpr)
				val.Args = append(val.Args, strparse.Expr(fmt.Sprintf("%s.%s", v.pkgName, v.FunName)))
			}
		}
	}
	return v
}
