package visitor

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// RefExpr builds an expression from the small references the generators emit,
// such as "NewUserBiz", "v1.NewUserRoute", "*cmd.MainApp" or "Repository[User]".
//
// Every node is created without position information. That matters: a node
// carrying offsets from a different file makes go/format break selectors and
// arguments apart at seemingly random places (loom.Graph[*App](loom.
//
//	Provide(NewApp)) ), because the printer trusts those offsets.
func RefExpr(text string) (ast.Expr, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("empty reference")
	}
	if strings.HasPrefix(text, "*") {
		inner, err := RefExpr(text[1:])
		if err != nil {
			return nil, err
		}
		return &ast.StarExpr{X: inner}, nil
	}

	base, typeArgs, err := splitTypeArgs(text)
	if err != nil {
		return nil, err
	}
	expr := dottedExpr(base)
	if len(typeArgs) == 0 {
		return expr, nil
	}

	args := make([]ast.Expr, 0, len(typeArgs))
	for _, arg := range typeArgs {
		parsed, err := RefExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, parsed)
	}
	if len(args) == 1 {
		return &ast.IndexExpr{X: expr, Index: args[0]}, nil
	}
	return &ast.IndexListExpr{X: expr, Indices: args}, nil
}

// splitTypeArgs separates "Repository[User]" into its name and type arguments.
func splitTypeArgs(text string) (string, []string, error) {
	open := strings.IndexByte(text, '[')
	if open < 0 {
		return text, nil, nil
	}
	if !strings.HasSuffix(text, "]") {
		return "", nil, fmt.Errorf("unbalanced type arguments in %q", text)
	}
	base := strings.TrimSpace(text[:open])
	inner := text[open+1 : len(text)-1]

	var args []string
	depth := 0
	start := 0
	for i, r := range inner {
		switch r {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, inner[start:i])
				start = i + 1
			}
		}
	}
	args = append(args, inner[start:])

	trimmed := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			return "", nil, fmt.Errorf("empty type argument in %q", text)
		}
		trimmed = append(trimmed, arg)
	}
	return base, trimmed, nil
}

// dottedExpr turns "a.b.c" into the selector chain a.b.c.
func dottedExpr(text string) ast.Expr {
	parts := strings.Split(text, ".")
	expr := ast.Expr(ast.NewIdent(parts[0]))
	for _, part := range parts[1:] {
		expr = &ast.SelectorExpr{X: expr, Sel: ast.NewIdent(part)}
	}
	return expr
}

// loomRef builds a reference into the loom package, such as loom.Provide.
func loomRef(name string) ast.Expr {
	return &ast.SelectorExpr{X: ast.NewIdent("loom"), Sel: ast.NewIdent(name)}
}

// loomCall builds a call into the loom package, such as loom.Name("AppGraph").
func loomCall(name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: loomRef(name), Args: args}
}

// stringLit builds a string literal without position information.
func stringLit(value string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", value)}
}
