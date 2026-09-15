package visitor

import (
	"bytes"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type FormatLine struct {
}

func NewFormatLine() *FormatLine {
	return &FormatLine{}
}

// FormatProvider puts one entry per line inside a dependency declaration:
// provider sets and loom.Graph literals alike.
//
// go/format only reflows whitespace it is given, so the multi-line layout of a
// generated provider.go has to be established here, otherwise appending a
// single constructor would reformat the whole set onto one long line.
func (v *FormatLine) FormatProvider(src any) ([]byte, error) {
	f, err := decorator.Parse(src)
	if err != nil {
		return nil, err
	}

	dst.Inspect(f, func(node dst.Node) bool {
		callExpr, ok := node.(*dst.CallExpr)
		if !ok || !isMultiLineCall(callExpr) {
			return true
		}
		for i := range callExpr.Args {
			callExpr.Args[i].Decorations().Before = dst.NewLine
			callExpr.Args[i].Decorations().After = dst.NewLine
		}
		return true
	})

	var dstBytes bytes.Buffer
	if err := decorator.Fprint(&dstBytes, f); err != nil {
		return nil, err
	}

	return dstBytes.Bytes(), nil
}

// isMultiLineCall reports whether call's arguments read better one per line:
// wire.NewSet, loom.Module and loom.Graph.
func isMultiLineCall(call *dst.CallExpr) bool {
	fun := call.Fun
	switch value := fun.(type) {
	case *dst.IndexExpr:
		fun = value.X
	case *dst.IndexListExpr:
		fun = value.X
	}
	selExpr, ok := fun.(*dst.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selExpr.X.(*dst.Ident)
	if !ok {
		return false
	}
	switch selExpr.Sel.Name {
	case "NewSet":
		return ident.Name == "wire"
	case "Module", "Graph":
		return ident.Name == "loom"
	default:
		return false
	}
}

func (v *FormatLine) FormatHttpEngine(src any) ([]byte, error) {
	f, err := decorator.Parse(src)
	if err != nil {
		return nil, err
	}

	dst.Inspect(f, func(node dst.Node) bool {
		switch n := node.(type) {

		// format in body initialize
		case *dst.KeyValueExpr:
			n.Decorations().Before = dst.NewLine
			n.Decorations().After = dst.NewLine
		case *dst.FuncDecl:
			if n.Name.Name != "NewHttpEngine" {
				return true
			}
			// format params
			var lists = n.Name.Obj.Decl.(*dst.FuncDecl).Type.Params.List
			for i := range lists {
				lists[i].Decorations().Before = dst.NewLine
				lists[i].Decorations().After = dst.NewLine
			}

		}

		return true
	})

	var dstBytes bytes.Buffer
	if err := decorator.Fprint(&dstBytes, f); err != nil {
		return nil, err
	}

	return dstBytes.Bytes(), nil
}
