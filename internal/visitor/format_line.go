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

// FormatProvider
func (v *FormatLine) FormatProvider(src any) ([]byte, error) {
	f, err := decorator.Parse(src)
	if err != nil {
		return nil, err
	}

	//if err := decorator.Print(f); err != nil {
	//	return nil, err
	//}

	// 目的：为NewSet参数添加换行
	dst.Inspect(f, func(node dst.Node) bool {
		if callExpr, ok := node.(*dst.CallExpr); ok {
			if selExpr, ok := callExpr.Fun.(*dst.SelectorExpr); ok {
				if ident, ok := selExpr.X.(*dst.Ident); ok {
					if ident.Name == "wire" && selExpr.Sel.Name == "NewSet" {
						if len(callExpr.Args) > 0 {
							for i := 0; i < len(callExpr.Args); i++ {
								callExpr.Args[i].Decorations().Before = dst.NewLine
								callExpr.Args[i].Decorations().After = dst.NewLine
							}
						}
					}
				}
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
			for i := 0; i < len(lists); i++ {
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
