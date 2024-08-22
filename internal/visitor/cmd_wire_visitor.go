package visitor

import (
	"go/ast"
)

type CmdWireVisitor struct {
	found bool

	CmdName      string
	UpperCmdName string
	AppName      string
}

func NewCmdWireVisitor(cmdName string, upperCmdName string, appName string) *CmdWireVisitor {
	return &CmdWireVisitor{CmdName: cmdName, UpperCmdName: upperCmdName, AppName: appName}
}

func (v *CmdWireVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return v
	}

	// 检查是否是函数声明节点并且函数名为 CmdName
	funcDecl, ok := node.(*ast.FuncDecl)
	if ok && funcDecl.Name.Name == v.CmdName {
		v.found = true
		return nil
	}

	return v
}

func (v *CmdWireVisitor) InsertInitCmdFunc(file *ast.File) {
	if !v.found {
		// 创建新的 InitCmd 函数代码块
		initCmdFunc := &ast.FuncDecl{
			Name: ast.NewIdent(v.UpperCmdName),
			Type: &ast.FuncType{
				Results: &ast.FieldList{
					List: []*ast.Field{
						{
							Type: ast.NewIdent("*" + v.AppName),
							Doc:  &ast.CommentGroup{List: []*ast.Comment{{Text: "// Return value 1"}}},
						},
						{
							Type: ast.NewIdent("func()"),
							Doc:  &ast.CommentGroup{List: []*ast.Comment{{Text: "// Return value 2"}}},
						},
						{
							Type: ast.NewIdent("error"),
							Doc:  &ast.CommentGroup{List: []*ast.Comment{{Text: "// Return value 3"}}},
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: ast.NewIdent("panic"),
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun: ast.NewIdent("wire.Build"),
									Args: []ast.Expr{
										ast.NewIdent("New" + v.AppName),
									},
								},
							},
						},
					},
				},
			},
		}

		// 将新函数添加到文件的声明列表中
		file.Decls = append(file.Decls, initCmdFunc)
	}
}
