package visitor

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

func UpdateProvider(f *ast.File, providerName, paramName string) {
	astutil.Apply(f, func(cursor *astutil.Cursor) bool {
		// 找到变量声明语句
		decl, ok := cursor.Node().(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR {
			return true
		}

		// 遍历变量声明的规范
		for _, spec := range decl.Specs {
			vspec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// 找到需要修改的变量声明
			if vspec.Names[0].Name != providerName {
				continue
			}

			// 在参数列表中添加新参数
			if len(vspec.Values) > 0 {
				callExpr, ok := vspec.Values[0].(*ast.CallExpr)
				if ok {
					// 获取原始参数列表
					args := callExpr.Args

					// 检查新参数是否已存在
					exists := false
					ast.Inspect(callExpr, func(node ast.Node) bool {
						if ident, ok := node.(*ast.Ident); ok && ident.Name == paramName {
							exists = true
							return false
						}
						return true
					})

					// 添加新参数
					if !exists {
						newParam := ast.NewIdent(paramName)
						args = append(args, newParam)
						callExpr.Args = args
					}
				}
			}
		}

		return true
	}, nil)
}
