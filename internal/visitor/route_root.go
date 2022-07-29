package visitor

import "go/ast"

/*
homeRoute *v1.HomeRoute

homeRoute: homeRoute,

r.homeRoute.Reg()
*/

type UpdateRoot struct {
	varName string //eg: homeRoute
	varType string //eg: *v1.HomeRoute
}

func NewUpdateRoot(varName string, varType string) *UpdateRoot {
	return &UpdateRoot{varName: varName, varType: varType}
}

func (v *UpdateRoot) Visit(node ast.Node) ast.Visitor {
	//fmt.Printf("%#v\n", node)

	switch n := node.(type) {
	//case *ast.TypeSpec:
	//	fmt.Printf("%#v\n", n)
	//	return v
	case *ast.StructType:
		//fmt.Printf("%#v\n", n)
		n.Fields.List = append(n.Fields.List,
			&ast.Field{
				Names: []*ast.Ident{ast.NewIdent(v.varName)},
				Type:  ast.NewIdent(v.varType),
			},
		)
		return v
	case *ast.CompositeLit:
		n.Elts = append(n.Elts,
			&ast.KeyValueExpr{
				Key:   ast.NewIdent(v.varName),
				Value: ast.NewIdent(v.varName),
			},
		)
		return v
	case *ast.FuncDecl:
		switch n.Name.Name {
		case "Register":
			n.Body.List = append(n.Body.List,
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "r"},
								Sel: &ast.Ident{Name: v.varName},
							},
							Sel: &ast.Ident{Name: "Reg"},
						},
					},
				},
			)

		case "NewHttpEngine":
			n.Type.Params.List = append(n.Type.Params.List,
				&ast.Field{
					Names: []*ast.Ident{ast.NewIdent(v.varName)},
					Type:  ast.NewIdent(v.varType),
				},
			)
		}
		return v
	}
	return v
}
