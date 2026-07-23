package route_info

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

// ─── Main entry point ────────────────────────────────────────────────────────

// AnalyzeRoutes scans a Go project rooted at projectRoot and extracts all
// HTTP route information by parsing Gin route registration patterns.
func AnalyzeRoutes(projectRoot string) (*ProjectRoutes, error) {
	ra := &routeAnalyzer{
		projectRoot:    projectRoot,
		importMapCache: map[string]map[string]string{},
		parsedCache:    map[string]*ast.File{},
		structCache:    map[string][]FieldInfo{},
		groupMapCache:  map[string]string{},
		resolvingTypes: map[string]bool{},
	}

	modName, err := readModuleName(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("read module name: %w", err)
	}
	ra.moduleName = modName

	if err := ra.scan(); err != nil {
		return nil, fmt.Errorf("scan routes: %w", err)
	}

	return &ProjectRoutes{
		Module: ra.moduleName,
		Routes: ra.routes,
	}, nil
}

// ─── Internal analyzer state ─────────────────────────────────────────────────

type routeAnalyzer struct {
	projectRoot string
	moduleName  string
	routes      []RouteInfo

	// cache: absolute file path → { import alias → full package path }
	importMapCache map[string]map[string]string
	// cache: absolute file path → parsed *ast.File
	parsedCache map[string]*ast.File
	// cache: "fullPkgPath.StructName" → fields
	structCache map[string][]FieldInfo
	// cache: file path → group var name → group prefix
	groupMapCache map[string]string
	// guard: set of cacheKey being resolved (prevents circular recursion)
	resolvingTypes map[string]bool
}

// ─── Scanning ────────────────────────────────────────────────────────────────

func (ra *routeAnalyzer) scan() error {
	// Route registration code is commonly, but not exclusively, placed under
	// routes/. Scan the project root so custom layouts are covered too; only
	// route structs are analyzed, keeping the additional work small.
	if err := ra.scanDir(ra.projectRoot, map[string]bool{}); err != nil {
		return fmt.Errorf("scan project: %w", err)
	}
	return nil
}

func (ra *routeAnalyzer) scanDir(dir string, scanned map[string]bool) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip middleware directories
			base := filepath.Base(path)
			if base == ".git" || base == ".agents" || base == "vendor" || base == "node_modules" ||
				base == "mdw" || base == "middleware" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if scanned[path] {
			return nil
		}
		scanned[path] = true

		if err := ra.analyzeFile(path); err != nil {
			log.Printf("warning: analyze %s: %v", relPath(ra.projectRoot, path), err)
		}
		return nil
	})
}

// ─── Per-file analysis ───────────────────────────────────────────────────────

func (ra *routeAnalyzer) analyzeFile(filePath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Build import map for this file.
	imports := buildImportMap(f)
	ra.importMapCache[filePath] = imports

	// Find route structs (struct types with "Route" suffix).
	var routeStructs []string
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if strings.HasSuffix(ts.Name.Name, "Route") {
				routeStructs = append(routeStructs, ts.Name.Name)
			}
		}
	}
	if len(routeStructs) == 0 {
		return nil // not a route file
	}

	// Collect all method declarations.
	methods := map[string]*ast.FuncDecl{}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fd.Recv == nil || len(fd.Recv.List) == 0 {
			continue
		}
		methods[fd.Name.Name] = fd
	}

	// For each route struct, analyze its Register() or legacy Reg() method.
	for _, structName := range routeStructs {
		regMethod := findRegMethod(structName, methods)
		if regMethod == nil {
			continue
		}
		ra.extractRegistrations(fset, filePath, structName, regMethod, methods, imports)
	}

	return nil
}

// ─── Find Reg() method for a given route struct ─────────────────────────────

func findRegMethod(structName string, methods map[string]*ast.FuncDecl) *ast.FuncDecl {
	for _, name := range []string{"Register", "Reg"} {
		reg, ok := methods[name]
		if ok && isMethodOfStruct(reg, structName) {
			return reg
		}
	}
	return nil
}

func isMethodOfStruct(fd *ast.FuncDecl, structName string) bool {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return false
	}
	field := fd.Recv.List[0]
	// Handle *StructName and StructName
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == structName
		}
	case *ast.Ident:
		return t.Name == structName
	}
	return false
}

// ─── Extract route registrations from Reg() body ────────────────────────────

func (ra *routeAnalyzer) extractRegistrations(
	fset *token.FileSet,
	filePath string,
	structName string,
	regMethod *ast.FuncDecl,
	methods map[string]*ast.FuncDecl,
	imports map[string]string,
) {
	if regMethod.Body == nil {
		return
	}

	// First pass: find group variable → prefix mappings.
	groupPrefix := map[string]string{} // var name → URL prefix
	for _, stmt := range regMethod.Body.List {
		ra.findGroupAssignments(stmt, groupPrefix)
	}

	// Second pass: find route registrations in all nested blocks.
	for _, stmt := range regMethod.Body.List {
		ra.extractFromBlock(fset, stmt, filePath, structName, groupPrefix, methods, imports)
	}
}

// findGroupAssignments detects group := r.g.Group("/prefix") patterns.
func (ra *routeAnalyzer) findGroupAssignments(stmt ast.Stmt, groupPrefix map[string]string) {
	ast.Inspect(stmt, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		ident, ok := as.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		if prefix := extractGroupPrefix(as.Rhs[0], groupPrefix); prefix != "" {
			groupPrefix[ident.Name] = prefix
		}
		return true
	})
}

// extractGroupPrefix extracts the URL prefix from r.g.Group("/prefix") chain.
func extractGroupPrefix(expr ast.Expr, groupPrefix map[string]string) string {
	// Walk through possible .Use() / .Group() call chains.
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}

	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	// Check for .Use(...) call chain:
	//   r.g.Group("/prefix").Use(...)
	if sel.Sel.Name == "Use" {
		// Recurse into the call receiver (the Group call)
		return extractGroupPrefix(sel.X, groupPrefix)
	}

	// Check for r.g.Group("/prefix")
	if sel.Sel.Name == "Group" {
		// First argument should be the path string literal.
		if len(callExpr.Args) == 0 {
			return ""
		}
		bl, ok := callExpr.Args[0].(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			return ""
		}
		prefix := strings.Trim(bl.Value, "\"")
		if parent, ok := sel.X.(*ast.Ident); ok {
			return combinePath(groupPrefix[parent.Name], prefix)
		}
		return combinePath(extractGroupPrefix(sel.X, groupPrefix), prefix)
	}

	return ""
}

// extractFromBlock recursively walks statements to find route registrations.
func (ra *routeAnalyzer) extractFromBlock(
	fset *token.FileSet,
	stmt ast.Stmt,
	filePath string,
	structName string,
	groupPrefix map[string]string,
	methods map[string]*ast.FuncDecl,
	imports map[string]string,
) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, inner := range s.List {
			ra.extractFromBlock(fset, inner, filePath, structName, groupPrefix, methods, imports)
		}

	case *ast.ExprStmt:
		ra.extractRouteRegistration(fset, s.X, filePath, structName, groupPrefix, methods, imports)

	case *ast.AssignStmt:
		// Also check for group-level calls inside assign statements like:
		//   _ = group  (to suppress unused variable)
		if len(s.Rhs) == 1 {
			if ce, ok := s.Rhs[0].(*ast.CallExpr); ok {
				ra.extractRouteRegistration(fset, ce, filePath, structName, groupPrefix, methods, imports)
			}
		}
	}
}

// extractRouteRegistration checks if an expression is a route registration
// like group.GET("/path", core.WrapData(r.handler())).
func (ra *routeAnalyzer) extractRouteRegistration(
	fset *token.FileSet,
	expr ast.Expr,
	filePath string,
	structName string,
	groupPrefix map[string]string,
	methods map[string]*ast.FuncDecl,
	imports map[string]string,
) {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}

	// The call should be: groupVar.HTTP_METHOD(args...)
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	httpMethod, pathIndex := routeMethodAndPathIndex(sel.Sel.Name, callExpr)
	if httpMethod == "" {
		return
	}

	// First argument should be a string literal (path).
	if len(callExpr.Args) <= pathIndex+1 {
		return
	}
	path := extractStringArg(callExpr, pathIndex)
	if path == "" && !isEmptyStringArg(callExpr, pathIndex) {
		return
	}

	// Second (or more) arguments contain the handler.
	// Look for core.WrapData(r.handler()) style.
	handlerName := ""

	handlerIndex := -1
	for i := pathIndex + 1; i < len(callExpr.Args); i++ {
		arg := callExpr.Args[i]
		if name, _ := extractHandlerName(arg); name != "" {
			handlerName = name
			handlerIndex = i
			break
		}
	}

	if handlerName == "" {
		return
	}

	// Determine group from the variable name.
	groupIdent, ok := sel.X.(*ast.Ident)
	groupType := "public"
	prefix := ""
	if ok {
		prefix = groupPrefix[groupIdent.Name]
		groupType = classifyGroup(prefix)
	}

	// Build full path by combining group prefix and route path.
	fullPath := combinePath(prefix, path)

	// Build absolute file path relative for output.
	rel := relPath(ra.projectRoot, filePath)

	info := RouteInfo{
		Method:   httpMethod,
		Path:     path,
		FullPath: fullPath,
		Handler:  handlerName,
		Group:    groupType,
		File:     rel,
		Line:     fset.Position(callExpr.Pos()).Line,
	}
	for i := pathIndex + 1; i < len(callExpr.Args); i++ {
		if i != handlerIndex {
			info.Middlewares = append(info.Middlewares, exprString(callExpr.Args[i]))
		}
	}

	// Find and analyze the handler method.
	handlerFuncDecl := findHandlerMethod(structName, handlerName, methods)
	if handlerFuncDecl != nil {
		// Find the inner handler closure and analyze it.
		innerBody := findInnerHandlerBody(handlerFuncDecl)
		if innerBody != nil {
			info.Params = ra.extractParams(innerBody, filePath, imports)
			// Build varTypes and pass to extractReturns for field resolution
			varTypes := ra.buildVarTypeMap(innerBody, filePath, imports)
			info.Returns = ra.extractReturns(innerBody, varTypes, filePath, imports)
		} else {
			info.Params, info.Returns = ra.extractTypedHandlerContract(handlerFuncDecl, filePath, imports, typedRequestSource(callExpr.Args[handlerIndex]))
		}
	}

	ra.routes = append(ra.routes, info)
}

// ─── Handler function resolution ─────────────────────────────────────────────

func structNameFromFile(filePath string, methods map[string]*ast.FuncDecl) string {
	// Find any method's receiver struct name.
	for _, fd := range methods {
		if fd.Recv == nil || len(fd.Recv.List) == 0 {
			continue
		}
		t := fd.Recv.List[0].Type
		switch tt := t.(type) {
		case *ast.StarExpr:
			if ident, ok := tt.X.(*ast.Ident); ok {
				return ident.Name
			}
		case *ast.Ident:
			return tt.Name
		}
	}
	return ""
}

func findHandlerMethod(structName, handlerName string, methods map[string]*ast.FuncDecl) *ast.FuncDecl {
	fd, ok := methods[handlerName]
	if !ok {
		return nil
	}
	if !isMethodOfStruct(fd, structName) {
		return nil
	}
	return fd
}

// extractHandlerName extracts a handler from legacy WrapData calls and the
// typed JSON/Request/NoInput wrappers used by current templates.
// e.g. core.WrapData(r.handlerName()) → "handlerName", []
// e.g. core.WrapData(r.handlerName(true)) → "handlerName", [true]
func extractHandlerName(expr ast.Expr) (string, []ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil
	}
	if sel.Sel.Name != "WrapData" && sel.Sel.Name != "JSON" && sel.Sel.Name != "Request" && sel.Sel.Name != "NoInput" {
		return "", nil
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", nil
	}
	_ = pkgIdent // could verify it's "core"

	if len(call.Args) == 0 {
		return "", nil
	}

	// Typed wrappers take r.handlerName; legacy WrapData takes r.handlerName().
	if handlerSel, ok := call.Args[0].(*ast.SelectorExpr); ok {
		if recvIdent, ok := handlerSel.X.(*ast.Ident); ok && recvIdent.Name == "r" {
			return handlerSel.Sel.Name, nil
		}
	}

	// The legacy argument should be r.handlerName(...)
	handlerCall, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		return "", nil
	}
	handlerSel, ok := handlerCall.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil
	}
	recvIdent, ok := handlerSel.X.(*ast.Ident)
	if !ok || recvIdent.Name != "r" {
		return "", nil
	}

	return handlerSel.Sel.Name, handlerCall.Args
}

func typedRequestSource(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "request"
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "request"
	}
	switch sel.Sel.Name {
	case "JSON":
		return "body"
	case "Request":
		return "request"
	default:
		return "request"
	}
}

func (ra *routeAnalyzer) extractTypedHandlerContract(fd *ast.FuncDecl, filePath string, imports map[string]string, source string) ([]ParamInfo, []ReturnInfo) {
	if fd.Type.Params == nil || fd.Type.Results == nil || len(fd.Type.Results.List) == 0 {
		return nil, nil
	}
	var params []ParamInfo
	// The first parameter is Gin context. Any following parameter is the
	// request contract in the current template.
	if len(fd.Type.Params.List) > 1 {
		typeStr := strings.TrimPrefix(exprString(fd.Type.Params.List[1].Type), "*")
		info := ParamInfo{Source: source, StructType: typeStr}
		if strings.Contains(typeStr, ".") {
			parts := strings.SplitN(typeStr, ".", 2)
			if pkg, ok := imports[parts[0]]; ok {
				info.Package = pkg
				info.Fields = ra.resolveStructFields(pkg, parts[1])
			}
		}
		params = append(params, info)
	}
	responseType := exprString(fd.Type.Results.List[0].Type)
	response := ReturnInfo{Type: responseType, Description: "success", Fields: ra.resolveReturnFields(responseType, filePath, imports)}
	return params, []ReturnInfo{response}
}

// findInnerHandlerBody finds the inner closure body from a handler method.
// The handler method has signature:
//
//	func (r *XxxRoute) handlerName() core.WrappedHandlerFunc {
//	    return func(c *gin.Context) (any, *core.RtnStatus) { ... }
//	}
func findInnerHandlerBody(fd *ast.FuncDecl) *ast.BlockStmt {
	if fd.Body == nil {
		return nil
	}
	for _, stmt := range fd.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		funcLit, ok := ret.Results[0].(*ast.FuncLit)
		if !ok {
			continue
		}
		if funcLit.Body != nil {
			return funcLit.Body
		}
	}
	return nil
}

// ─── Parameter extraction ────────────────────────────────────────────────────

func (ra *routeAnalyzer) extractParams(body *ast.BlockStmt, filePath string, imports map[string]string) []ParamInfo {
	var params []ParamInfo
	seen := map[string]bool{} // dedup by "source:key" or "source:struct"

	// Build a variable type map from declarations in the body.
	varTypes := ra.buildVarTypeMap(body, filePath, imports)

	ast.Inspect(body, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		receiverIdent, ok := sel.X.(*ast.Ident)
		if !ok || receiverIdent.Name != "c" {
			return true
		}

		methodName := sel.Sel.Name
		switch methodName {
		case "ShouldBindJSON", "BindJSON", "ShouldBindXML", "BindXML", "ShouldBindYAML", "BindYAML":
			if info := ra.extractBindParam(callExpr, "body", varTypes, filePath, imports); info != nil {
				key := "body:" + info.StructType
				if !seen[key] {
					params = append(params, *info)
					seen[key] = true
				}
			}

		case "ShouldBindQuery", "BindQuery":
			if info := ra.extractBindParam(callExpr, "query", varTypes, filePath, imports); info != nil {
				key := "query:" + info.StructType
				if !seen[key] {
					params = append(params, *info)
					seen[key] = true
				}
			}

		case "ShouldBindUri", "BindUri":
			if info := ra.extractBindParam(callExpr, "uri", varTypes, filePath, imports); info != nil {
				key := "uri:" + info.StructType
				if !seen[key] {
					params = append(params, *info)
					seen[key] = true
				}
			}

		case "ShouldBind", "Bind":
			if info := ra.extractBindParam(callExpr, "request", varTypes, filePath, imports); info != nil {
				key := "request:" + info.StructType
				if !seen[key] {
					params = append(params, *info)
					seen[key] = true
				}
			}

		case "Query":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "query", Key: key, Type: "string"}
				key2 := "query:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "GetQuery":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "query", Key: key, Type: "string"}
				key2 := "query:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "QueryArray", "GetQueryArray":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "query", Key: key, Type: "[]string"}
				key2 := "query:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "DefaultQuery":
			if key := extractStringArg(callExpr, 0); key != "" {
				def := extractStringArg(callExpr, 1)
				info := ParamInfo{Source: "query", Key: key, Type: "string", Default: def}
				key2 := "query:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "Param":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "uri", Key: key, Type: "string"}
				key2 := "uri:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "Get":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "context", Key: key}
				key2 := "context:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "GetHeader":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "header", Key: key, Type: "string"}
				key2 := "header:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "PostForm", "GetPostForm", "DefaultPostForm":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "form", Key: key, Type: "string"}
				if methodName == "DefaultPostForm" {
					info.Default = extractStringArg(callExpr, 1)
				}
				key2 := "form:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}

		case "FormFile":
			if key := extractStringArg(callExpr, 0); key != "" {
				info := ParamInfo{Source: "file", Key: key, Type: "file"}
				key2 := "file:" + key
				if !seen[key2] {
					params = append(params, info)
					seen[key2] = true
				}
			}
		}

		return true
	})

	return params
}

// buildVarTypeMap builds a map of variable name → type expression string
// from declarations in the handler body.
func (ra *routeAnalyzer) buildVarTypeMap(body *ast.BlockStmt, filePath string, imports map[string]string) map[string]string {
	varTypes := map[string]string{}

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.DeclStmt:
			gd, ok := node.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) == 0 {
					continue
				}
				if vs.Type != nil {
					varTypes[vs.Names[0].Name] = exprString(vs.Type)
				}
			}

		case *ast.AssignStmt:
			if node.Tok != token.DEFINE {
				return true
			}

			// Handle multi-assignments like payload, err := fn()
			// Map each LHS ident if there's a single RHS (common pattern).
			if len(node.Rhs) == 1 {
				rhs := node.Rhs[0]
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						if _, exists := varTypes[ident.Name]; !exists {
							varTypes[ident.Name] = rhsTypeString(rhs)
						}
					}
				}
			} else if len(node.Lhs) == len(node.Rhs) {
				// 1:1 mapping: a, b := expr1, expr2
				for i, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						if _, exists := varTypes[ident.Name]; !exists {
							varTypes[ident.Name] = rhsTypeString(node.Rhs[i])
						}
					}
				}
			}
		}
		return true
	})

	return varTypes
}

// extractBindParam extracts param info from c.ShouldBindJSON(&pm) or
// c.ShouldBindQuery(&pm) calls.
func (ra *routeAnalyzer) extractBindParam(
	callExpr *ast.CallExpr,
	source string,
	varTypes map[string]string,
	filePath string,
	imports map[string]string,
) *ParamInfo {
	if len(callExpr.Args) == 0 {
		return nil
	}

	// Argument is &pm
	unary, ok := callExpr.Args[0].(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return nil
	}

	ident, ok := unary.X.(*ast.Ident)
	if !ok {
		return nil
	}

	varName := ident.Name
	typeStr, ok := varTypes[varName]
	if !ok {
		// Try direct var decl again
		return &ParamInfo{Source: source, StructType: "unknown"}
	}

	info := ParamInfo{
		Source:     source,
		StructType: typeStr,
	}

	// Try to resolve the struct fields.
	if strings.Contains(typeStr, ".") {
		// Qualified type like "params.LoginParams"
		parts := strings.SplitN(typeStr, ".", 2)
		pkgAlias := parts[0]
		typeName := parts[1]

		if fullPath, ok := imports[pkgAlias]; ok {
			info.Package = fullPath
			fields := ra.resolveStructFields(fullPath, typeName)
			if fields != nil {
				info.Fields = fields
			}
		}
	} else {
		// Unqualified type — might be in the same package.
		fields := ra.resolveStructFields("", typeStr)
		if fields != nil {
			info.Fields = fields
		}
	}

	return &info
}

// ─── Return extraction ───────────────────────────────────────────────────────

func (ra *routeAnalyzer) extractReturns(body *ast.BlockStmt, varTypes map[string]string, filePath string, imports map[string]string) []ReturnInfo {
	var returns []ReturnInfo
	seen := map[string]bool{}

	ast.Inspect(body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}

		info := ra.buildReturnInfo(ret.Results, varTypes, filePath, imports)

		if info.Type != "" {
			key := info.Type + ":" + info.Description
			if !seen[key] {
				returns = append(returns, info)
				seen[key] = true
			}
		}

		return true
	})

	return returns
}

// buildReturnInfo constructs a ReturnInfo from a return statement's results.
// It resolves variable names to their types and tries to populate Fields.
func (ra *routeAnalyzer) buildReturnInfo(results []ast.Expr, varTypes map[string]string, filePath string, imports map[string]string) ReturnInfo {
	if len(results) == 2 {
		first := results[0]
		second := results[1]

		// return nil, core.NewRtnWithErr(err) → error
		if isNil(first) && isNewRtnWithErr(second) {
			return ReturnInfo{Type: "error", Description: "error"}
		}
		// return nil, core.NewRtnStatus(...) → error with custom code
		if isNil(first) && isNewRtnStatus(second) {
			return ReturnInfo{Type: "error", Description: "custom error status"}
		}
		// return data, nil → success
		if isNil(second) || isCoreSuccess(second) {
			return ra.returnInfoFromExpr(first, varTypes, filePath, imports, "success")
		}
		// return data, core.NewRtnStatus(...) → custom status
		if isNewRtnStatus(second) {
			return ra.returnInfoFromExpr(first, varTypes, filePath, imports, "custom status")
		}
		// return data, core.NewRtnWithErr(...) → error with data
		if isNewRtnWithErr(second) {
			return ra.returnInfoFromExpr(first, varTypes, filePath, imports, "error")
		}
		// fallback
		return ra.returnInfoFromExpr(first, varTypes, filePath, imports, "success")
	}

	if len(results) == 1 {
		first := results[0]
		if isNewListRtn(first) {
			return ReturnInfo{Type: "paginated_list", Description: "list response"}
		}
		if isNewRtnWithErr(first) {
			return ReturnInfo{Type: "error", Description: "error"}
		}
		return ra.returnInfoFromExpr(first, varTypes, filePath, imports, "response")
	}

	return ReturnInfo{}
}

// returnInfoFromExpr builds a ReturnInfo from a single expression,
// resolving variable names to their full type and field details.
func (ra *routeAnalyzer) returnInfoFromExpr(expr ast.Expr, varTypes map[string]string, filePath string, imports map[string]string, desc string) ReturnInfo {
	typeStr := returnExprType(expr)
	info := ReturnInfo{Type: typeStr, Description: desc}

	// If the expression is a variable name, look up its type.
	if ident, ok := expr.(*ast.Ident); ok && !isNil(expr) {
		if resolved, found := varTypes[ident.Name]; found {
			info.Type = resolved
			info.Fields = ra.resolveReturnFields(resolved, filePath, imports)
		}
		return info
	}

	// If it's a call expression like gin.H{...}, the type was captured above.
	// Try to resolve "gin.H" etc.
	if _, ok := expr.(*ast.CallExpr); ok {
		info.Fields = ra.resolveReturnFields(typeStr, filePath, imports)
		return info
	}

	// CompositeLit: extract keys from the literal itself.
	if cl, ok := expr.(*ast.CompositeLit); ok {
		resolved := exprString(cl.Type)
		if resolved != "" {
			info.Type = resolved
			info.Fields = ra.resolveReturnFields(resolved, filePath, imports)
		}
		// If struct fields weren't resolvable (e.g. gin.H from external pkg),
		// extract the literal keys as pseudo-fields.
		if len(info.Fields) == 0 && cl.Elts != nil {
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				keyStr := extractCompositeLitKey(kv.Key)
				if keyStr != "" {
					valType := returnExprType(kv.Value)
					info.Fields = append(info.Fields, FieldInfo{
						Name: keyStr,
						Type: valType,
					})
				}
			}
		}
		return info
	}

	// SelectorExpr: e.g., core.Success → we already handled this upstream.
	// Return directly.
	return info
}

// resolveReturnFields tries to resolve struct fields from a type string.
// The typeStr can be "params.LoginParams", "*params.LoginParams", "gin.H", etc.
func (ra *routeAnalyzer) resolveReturnFields(typeStr string, filePath string, imports map[string]string) []FieldInfo {
	// Strip leading * or []
	clean := strings.TrimLeft(typeStr, "*[]")
	if clean == typeStr && strings.HasPrefix(typeStr, "map[") {
		// map types: skip field resolution
		return nil
	}

	if clean == typeStr && strings.HasPrefix(typeStr, "[]") {
		clean = strings.TrimPrefix(typeStr, "[]")
	}

	// Check for dotted type: params.LoginParams
	if strings.Contains(clean, ".") {
		parts := strings.SplitN(clean, ".", 2)
		pkgAlias := parts[0]
		typeName := parts[1]
		if fullPath, ok := imports[pkgAlias]; ok {
			return ra.resolveStructFields(fullPath, typeName)
		}
		// Also check in the file's own package.
		return ra.resolveStructFields("", typeName)
	}

	// Unqualified type name — maybe same package.
	return ra.resolveStructFields("", clean)
}

// ─── Struct type resolution ──────────────────────────────────────────────────

// resolveStructFields tries to find and parse a struct definition.
// fullPkgPath is the full import path (empty for same-package types).
// typeName is the struct name.
func (ra *routeAnalyzer) resolveStructFields(fullPkgPath, typeName string) []FieldInfo {
	cacheKey := fullPkgPath + "." + typeName
	if fields, ok := ra.structCache[cacheKey]; ok {
		return fields
	}
	// Guard: prevent re-entering resolution for the same type (circular refs).
	if ra.resolvingTypes[cacheKey] {
		return nil
	}
	ra.resolvingTypes[cacheKey] = true
	defer delete(ra.resolvingTypes, cacheKey)

	// Determine the directory for the package.
	pkgDir := ""
	if fullPkgPath == "" || fullPkgPath == ra.moduleName {
		// Same package or root module — search in already-parsed files.
		// We'll need to check all parsed files for the type.
		for _, pf := range ra.parsedCache {
			pkg := pf.Name.Name
			if typeFields := ra.findStructInFile(pf, typeName); typeFields != nil {
				ra.structCache[cacheKey] = typeFields
				ra.structCache[pkg+"."+typeName] = typeFields
				return typeFields
			}
		}
		return nil
	}

	// For external or local packages, try to find the directory.
	if after, ok := strings.CutPrefix(fullPkgPath, ra.moduleName); ok {
		// Local project package.
		rel := after
		rel = strings.TrimPrefix(rel, "/")
		pkgDir = filepath.Join(ra.projectRoot, rel)
	} else {
		// External package — skip for now.
		return nil
	}

	if pkgDir == "" {
		return nil
	}

	// Parse all Go files in the package directory.
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		fp := filepath.Join(pkgDir, entry.Name())
		fset := token.NewFileSet()
		pf, err := parser.ParseFile(fset, fp, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		ra.parsedCache[fp] = pf

		if fields := ra.findStructInFile(pf, typeName); fields != nil {
			ra.structCache[cacheKey] = fields
			return fields
		}
	}

	return nil
}

func (ra *routeAnalyzer) findStructInFile(f *ast.File, typeName string) []FieldInfo {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != typeName {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			return ra.extractFieldsFromStruct(st)
		}
	}
	return nil
}

func (ra *routeAnalyzer) extractFieldsFromStruct(st *ast.StructType) []FieldInfo {
	if st.Fields == nil {
		return nil
	}
	var fields []FieldInfo
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			// Embedded field — skip.
			continue
		}
		for _, name := range f.Names {
			typeStr := exprString(f.Type)
			fi := FieldInfo{
				Name: name.Name,
				Type: typeStr,
			}
			if f.Tag != nil {
				tagRaw := strings.Trim(f.Tag.Value, "`")
				fi.Tag = tagRaw
				if bindingTagRequired(tagRaw) {
					fi.Required = true
				}
			}
			// Try to resolve nested struct type — strip slice/pointer wrappers.
			nestedType := strings.TrimLeft(typeStr, "*[]")
			if after, ok := strings.CutPrefix(typeStr, "[]"); ok {
				nestedType = after
			}
			nestedType = strings.TrimLeft(nestedType, "*")

			// Skip simple/built-in types.
			if isBuiltinType(nestedType) {
				fields = append(fields, fi)
				continue
			}

			if childFields := ra.resolveNestedTypeName(nestedType); childFields != nil {
				fi.Fields = childFields
			}
			fields = append(fields, fi)
		}
	}
	return fields
}

// bindingTagRequired reports whether the Gin binding rule list contains the
// required rule. Binding tags frequently include additional validators, e.g.
// binding:"required,alphanum,min=6", so an exact tag match is insufficient.
func bindingTagRequired(tag string) bool {
	for rule := range strings.SplitSeq(reflect.StructTag(tag).Get("binding"), ",") {
		if strings.TrimSpace(rule) == "required" {
			return true
		}
	}
	return false
}

// resolveNestedTypeName resolves a qualified type name like "params.SyncDiskTagItem"
// by trying all available import maps from parsed files.
func (ra *routeAnalyzer) resolveNestedTypeName(typeName string) []FieldInfo {
	if !strings.Contains(typeName, ".") {
		return ra.resolveStructFields("", typeName)
	}
	parts := strings.SplitN(typeName, ".", 2)
	pkgAlias := parts[0]
	structName := parts[1]

	// Search through all parsed files' import maps.
	for _, pf := range ra.parsedCache {
		imports := buildImportMap(pf)
		if fullPath, ok := imports[pkgAlias]; ok {
			if fields := ra.resolveStructFields(fullPath, structName); fields != nil {
				return fields
			}
		}
		// Also try same-package resolution from this file.
		if pf.Name.Name == pkgAlias {
			if fields := ra.resolveStructFields("", structName); fields != nil {
				return fields
			}
		}
	}
	// Also try directly from already cached types.
	if fields := ra.resolveStructFields("", structName); fields != nil {
		return fields
	}
	return nil
}

// ─── Utility helpers ─────────────────────────────────────────────────────────

// isHTTPMethod returns true if the string is an HTTP method.
func isHTTPMethod(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return true
	}
	return false
}

// routeMethodAndPathIndex recognizes Gin's verb helpers as well as the
// generic Any and Handle registrations. Handle keeps a literal method when it
// is statically available; otherwise it is reported as HANDLE.
func routeMethodAndPathIndex(name string, call *ast.CallExpr) (string, int) {
	if isHTTPMethod(name) {
		return name, 0
	}
	switch name {
	case "Any":
		return "ANY", 0
	case "Handle":
		if method := extractStringArg(call, 0); method != "" {
			return strings.ToUpper(method), 1
		}
		return "HANDLE", 1
	}
	return "", 0
}

func isEmptyStringArg(ce *ast.CallExpr, idx int) bool {
	if idx >= len(ce.Args) {
		return false
	}
	bl, ok := ce.Args[idx].(*ast.BasicLit)
	return ok && bl.Kind == token.STRING && strings.Trim(bl.Value, "\"") == ""
}

// isBuiltinType returns true for Go built-in types that are never structs.
func isBuiltinType(t string) bool {
	switch t {
	case "string", "bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune", "error", "any":
		return true
	}
	return false
}

// combinePath joins a group prefix and a route path, normalizing slashes.
func combinePath(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "" || path == "/" {
		return prefix
	}
	// Ensure prefix ends without trailing slash, path starts without leading slash.
	prefix = strings.TrimRight(prefix, "/")
	path = strings.TrimLeft(path, "/")
	return prefix + "/" + path
}

// classifyGroup categorizes a URL prefix into a group type.
func classifyGroup(prefix string) string {
	if strings.Contains(prefix, "/admin/") {
		return "admin"
	}
	if strings.Contains(prefix, "/auth/") {
		return "auth"
	}
	return "public"
}

// buildImportMap builds a map of import alias → full package path.
func buildImportMap(f *ast.File) map[string]string {
	m := map[string]string{}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			m[imp.Name.Name] = path
		} else {
			// Default alias is the last segment of the path.
			parts := strings.Split(path, "/")
			alias := parts[len(parts)-1]
			m[alias] = path
		}
	}
	return m
}

// readModuleName reads the module name from go.mod.
func readModuleName(projectRoot string) (string, error) {
	gp := filepath.Join(projectRoot, "go.mod")
	data, err := os.ReadFile(gp)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`(?m)^module\s+(\S+)`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) < 2 {
		return "", fmt.Errorf("module name not found in go.mod")
	}
	return matches[1], nil
}

// relPath returns a relative path from base to target.
func relPath(base, target string) string {
	r, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return r
}

// exprString converts an AST expression to a readable string.
func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprString(e.Elt)
		}
		return "[" + exprString(e.Len) + "]" + exprString(e.Elt)
	case *ast.MapType:
		return "map[" + exprString(e.Key) + "]" + exprString(e.Value)
	case *ast.BasicLit:
		return e.Value
	case *ast.InterfaceType:
		return "any"
	case *ast.FuncType:
		return "func(...)"
	case *ast.StructType:
		return "struct{...}"
	case *ast.CompositeLit:
		return exprString(e.Type)
	case *ast.CallExpr:
		return exprString(e.Fun) + "(...)"
	default:
		return fmt.Sprintf("%T", e)
	}
}

// extractStringArg extracts a string literal argument from a call expression.
func extractStringArg(ce *ast.CallExpr, idx int) string {
	if idx >= len(ce.Args) {
		return ""
	}
	bl, ok := ce.Args[idx].(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return ""
	}
	return strings.Trim(bl.Value, "\"")
}

// isNil checks if an expression is the nil literal.
func isNil(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

// isCoreSuccess checks if expression is core.Success.
func isCoreSuccess(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "core" && sel.Sel.Name == "Success"
}

// isNewRtnWithErr checks if expression is core.NewRtnWithErr(...).
func isNewRtnWithErr(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "core" && sel.Sel.Name == "NewRtnWithErr"
}

// isNewRtnStatus checks if expression is core.NewRtnStatus(...).
func isNewRtnStatus(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "core" && sel.Sel.Name == "NewRtnStatus"
}

// isNewListRtn checks if expression is core.NewListRtn(...).
func isNewListRtn(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "core" && sel.Sel.Name == "NewListRtn"
}

// returnExprType returns a string representation of a return expression's type.
func returnExprType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "nil" {
			return "nil"
		}
		// Variable — try to be descriptive
		return e.Name
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			return "string"
		}
		return e.Kind.String()
	case *ast.CallExpr:
		return exprString(e.Fun)
	case *ast.CompositeLit:
		return exprString(e.Type)
	case *ast.SelectorExpr:
		return exprString(e)
	case *ast.UnaryExpr:
		return exprString(e.X)
	default:
		// Fallback: try to get a readable string
		s := exprString(expr)
		if s != "" {
			return s
		}
		return fmt.Sprintf("%T", expr)
	}
}

// rhsTypeString tries to extract a type string from a RHS expression.
func rhsTypeString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return exprString(e.Type)
	case *ast.CallExpr:
		return exprString(e.Fun)
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return rhsTypeString(e.X)
		}
	case *ast.Ident:
		return e.Name
	}
	return exprString(expr)
}

// extractCompositeLitKey extracts a string key from a composite literal key expression.
// Handles "key" (string literal) and identifiers.
func extractCompositeLitKey(key ast.Expr) string {
	switch k := key.(type) {
	case *ast.BasicLit:
		if k.Kind == token.STRING {
			return strings.Trim(k.Value, "\"")
		}
	case *ast.Ident:
		return k.Name
	}
	return ""
}
