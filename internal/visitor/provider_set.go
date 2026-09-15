package visitor

import (
	"go/ast"
	"go/token"
)

// AppendProvider appends a constructor to the named provider set declaration,
// returning false when the set already contains it.
//
// The set may be either a Loom module or a legacy Wire set, and the entry is
// rendered to match:
//
//	var ProviderBizSet = loom.Module(loom.Provide(NewUserBiz))     // appended as loom.Provide(NewOrderBiz)
//	var ProviderBizSet = wire.NewSet(NewUserBiz)                   // appended as NewOrderBiz
//
// Supporting both keeps `nr gen` working on projects that have not run
// `nr loom convert` yet.
func AppendProvider(f *ast.File, setName, ctor string) bool {
	call := findProviderSet(f, setName)
	if call == nil {
		return false
	}
	for _, arg := range call.Args {
		if providerArgName(arg) == ctor {
			return true
		}
	}
	ref, err := RefExpr(ctor)
	if err != nil {
		return false
	}
	if isLoomModule(call) {
		call.Args = append(call.Args, loomCall("Provide", ref))
		return true
	}
	call.Args = append(call.Args, ref)
	return true
}

// findProviderSet returns the call expression initialising the named package
// level variable, or nil when no such declaration exists.
func findProviderSet(f *ast.File, name string) *ast.CallExpr {
	var found *ast.CallExpr
	ast.Inspect(f, func(node ast.Node) bool {
		if found != nil {
			return false
		}
		decl, ok := node.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != name || len(valueSpec.Values) != 1 {
				continue
			}
			call, ok := valueSpec.Values[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			found = call
			return false
		}
		return true
	})
	return found
}

// isLoomModule reports whether call is a loom.Module(...) invocation, whose
// arguments are loom.Option values rather than bare constructors.
func isLoomModule(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Module"
}

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

// providerArgName returns the constructor reference inside one provider set
// argument, unwrapping loom.Provide(ctor) and loom.As[I](ctor) so a set that
// already binds a constructor is not extended with a duplicate entry.
func providerArgName(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return exprString(expr)
	}
	fun := unwrapIndex(call.Fun)
	selector, ok := fun.(*ast.SelectorExpr)
	if !ok || (selector.Sel.Name != "Provide" && selector.Sel.Name != "As") || len(call.Args) != 1 {
		return exprString(expr)
	}
	return exprString(call.Args[0])
}
