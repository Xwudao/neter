package visitor

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

// GraphSpec describes one loom.Graph declaration to add to a file.
type GraphSpec struct {
	// VarName is the package level variable holding the graph, such as
	// migrateAppGraph. It must not collide with an existing declaration.
	VarName string
	// Injector is the name of the generated constructor, such as MigrateCmd.
	// It is passed to loom.Name so call sites keep the name they already use.
	Injector string
	// Target is the graph's result type, such as *MigrateApp.
	Target string
	// Providers are the constructor references listed in the graph, such as
	// NewMigrateApp or seed.NewRegistry.
	Providers []string
	// Imports are added to the file when missing.
	Imports []string
}

// AddGraph appends a loom.Graph declaration for a generated command.
//
// The injector name is preserved with loom.Name so call sites keep using
// cmd_app.<Name>Cmd() unchanged, and the graph variable is derived from it
// (<name>CmdGraph) because a Go package cannot hold a variable and a function
// with the same name.
//
// The declaration is inserted into the existing var (...) block when the file
// already declares a graph, so repeated generation keeps one readable block
// instead of a pile of separate var statements.
func AddGraph(fset *token.FileSet, f *ast.File, spec GraphSpec) error {
	if spec.VarName == "" || spec.Injector == "" || spec.Target == "" {
		return fmt.Errorf("graph spec needs a variable name, injector and target type")
	}
	if hasDecl(f, spec.VarName) {
		return fmt.Errorf("%s already exists in this file", spec.VarName)
	}
	for _, path := range spec.Imports {
		astutil.AddImport(fset, f, path)
	}

	target, err := RefExpr(spec.Target)
	if err != nil {
		return fmt.Errorf("graph target %q: %w", spec.Target, err)
	}
	args := []ast.Expr{loomCall("Name", stringLit(spec.Injector))}
	for _, provider := range spec.Providers {
		providerRef, err := RefExpr(provider)
		if err != nil {
			return fmt.Errorf("graph provider %q: %w", provider, err)
		}
		args = append(args, loomCall("Provide", providerRef))
	}
	graph := &ast.IndexExpr{X: loomRef("Graph"), Index: target}
	valueSpec := &ast.ValueSpec{
		Names:  []*ast.Ident{ast.NewIdent(spec.VarName)},
		Values: []ast.Expr{&ast.CallExpr{Fun: graph, Args: args}},
	}

	if block := graphBlock(f); block != nil {
		block.Specs = append(block.Specs, valueSpec)
		return nil
	}
	f.Decls = append(f.Decls, &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{valueSpec}})
	return nil
}

// graphBlock returns the var declaration that new graphs should join.
//
// A bare `var appGraph = loom.Graph[...](...)` is upgraded to a parenthesised
// block on first append, so repeatedly generating commands converges on one
// readable declaration instead of a growing pile of var statements.
func graphBlock(f *ast.File) *ast.GenDecl {
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR || !declaresGraph(genDecl) {
			continue
		}
		if !genDecl.Lparen.IsValid() {
			genDecl.Lparen = genDecl.Pos()
			genDecl.Rparen = genDecl.End()
		}
		return genDecl
	}
	return nil
}

func declaresGraph(genDecl *ast.GenDecl) bool {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, value := range valueSpec.Values {
			if call, ok := value.(*ast.CallExpr); ok && isLoomGraph(call) {
				return true
			}
		}
	}
	return false
}

// isLoomGraph reports whether call is loom.Graph[T](...). The type argument
// makes the callee an index expression rather than a selector, so the selector
// has to be unwrapped first.
func isLoomGraph(call *ast.CallExpr) bool {
	selector, ok := unwrapIndex(call.Fun).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Graph" {
		return false
	}
	if pkg, ok := selector.X.(*ast.Ident); ok {
		return pkg.Name == "loom"
	}
	return false
}

// unwrapIndex removes a generic instantiation, so `pkg.Fn[T]` and `pkg.Fn` yield
// the same `pkg.Fn` selector.
func unwrapIndex(expr ast.Expr) ast.Expr {
	switch value := expr.(type) {
	case *ast.IndexExpr:
		return value.X
	case *ast.IndexListExpr:
		return value.X
	default:
		return expr
	}
}

// hasDecl reports whether the file declares a package level identifier.
func hasDecl(f *ast.File, name string) bool {
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			switch value := spec.(type) {
			case *ast.ValueSpec:
				for _, ident := range value.Names {
					if ident.Name == name {
						return true
					}
				}
			case *ast.TypeSpec:
				if value.Name.Name == name {
					return true
				}
			}
		}
	}
	return false
}
