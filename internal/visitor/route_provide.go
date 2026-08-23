package visitor

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/go-toolsmith/strparse"
)

// RouteProvideVisitor appends one constructor to ProviderRouteSet. It only
// mutates the named wire.NewSet declaration, avoiding the previous assumption
// that the first var declaration in provider.go was the route provider set.
type RouteProvideVisitor struct {
	pkgName string
	funName string
	updated bool
}

func NewRouteProvideVisitor(pkgName string, funName string) *RouteProvideVisitor {
	return &RouteProvideVisitor{pkgName: pkgName, funName: funName}
}

func (v *RouteProvideVisitor) Visit(node ast.Node) ast.Visitor {
	decl, ok := node.(*ast.GenDecl)
	if !ok || decl.Tok != token.VAR || v.updated {
		return v
	}
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "ProviderRouteSet" || len(valueSpec.Values) != 1 {
			continue
		}
		call, ok := valueSpec.Values[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		for _, arg := range call.Args {
			if exprString(arg) == v.pkgName+"."+v.funName {
				v.updated = true
				return v
			}
		}
		call.Args = append(call.Args, strparse.Expr(fmt.Sprintf("%s.%s", v.pkgName, v.funName)))
		v.updated = true
		return v
	}
	return v
}

func (v *RouteProvideVisitor) Updated() bool { return v.updated }

func exprString(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return exprString(value.X) + "." + value.Sel.Name
	default:
		return ""
	}
}
