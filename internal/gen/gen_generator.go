package gen

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/internal/tpl"
	"github.com/Xwudao/neter/internal/visitor"
	"github.com/Xwudao/neter/pkg/utils"
)

type Generator struct {
	RootPath        string
	RouteNameSuffix string
	PackageName     string
	ModName         string
	Pkg             string // explicit package/subdir (CLI mode)

	WithCRUD      bool
	WithIface     bool // generate _biz_iface.go + mockgen directive
	WithContracts bool // generate business Command/Query contracts
	EntName       string
	Model         string // sqlc model name (e.g. User)
	Plural        string // sqlc list method plural (e.g. Users)
	Table         string // sqlc table name (e.g. users)
	IsSQLC        bool   // project uses the new PostgreSQL + sqlc stack
	V2            bool

	routeTpl       string
	bizTpl         string
	bizIfaceTpl    string
	repoTpl        string
	bizParamsTpl   string
	bizContractTpl string

	bizSQLCTpl      string
	bizIfaceSQLCTpl string
	repoSQLCTpl     string

	FilenameRouteSuffix string
	FilenameBizSuffix   string
	FilenameRepoSuffix  string

	saveRouteFilePath       string
	saveBizFilePath         string
	saveBizIfaceFilePath    string
	saveRepoFilePath        string
	saveBizParamsFilePath   string
	saveBizContractFilePath string

	Name     string
	TypeName string

	StructRouteName string
	StructBizName   string
	StructRepoName  string

	// UseRouteRegistry is enabled by templates that expose a central
	// NewRouteRegistry function.  Older projects keep their existing root.go
	// mutation behaviour unchanged.
	UseRouteRegistry bool
	// UseTypedAPI is separate from route registration. A migrated legacy
	// project can use RouteRegistry while continuing to generate WrapData
	// handlers until its API contracts are migrated.
	UseTypedAPI bool
	// UseErrorTypedAPI selects the modern JSONE/RequestE/NoInputE contract
	// whose handlers return error instead of *core.RtnStatus.
	UseErrorTypedAPI bool
	// UseRouterRegister is enabled after the optional router-injection
	// migration. Older registry projects continue to receive g-backed routes.
	UseRouterRegister bool
}

type Request struct {
	TypeName      string
	Name          string
	Pkg           string // route sub-package (e.g. "v1"); required for route in CLI mode
	NoRepo        bool
	WithCRUD      bool
	WithParams    bool
	WithIface     bool // generate a _biz_iface.go file and add a mockgen directive
	WithContracts bool // generate Command/Query types for transport-neutral biz APIs
	EntName       string
	Model         string // sqlc model name (e.g. User); alias of EntName on new projects
	Plural        string // sqlc list method plural (default <Model>s)
	V2            bool
	SkipWire      bool
}

func NewGenerator(req Request) *Generator {
	return &Generator{
		Name:          req.Name,
		TypeName:      req.TypeName,
		Pkg:           req.Pkg,
		WithCRUD:      req.WithCRUD,
		WithIface:     req.WithIface,
		WithContracts: req.WithContracts,
		EntName:       req.EntName,
		Model:         req.Model,
		Plural:        req.Plural,
		V2:            req.V2,
	}
}

func Execute(req Request) (err error) {
	g := NewGenerator(req)
	if err := g.prepare(); err != nil {
		return err
	}
	// Generating a route or biz changes several files. Snapshot them before
	// mutation so a later registry/provider/Wire failure does not strand the
	// project in a half-generated state.
	rollback, err := g.snapshotMutationFiles()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			return
		}
		if rollbackErr := rollback(); rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback generated files: %w", rollbackErr))
		}
	}()

	switch req.TypeName {
	case "route":
		if err := g.GenRoute(); err != nil {
			return err
		}
		if g.UseRouteRegistry {
			if err := g.updateRouteRegistry(); err != nil {
				return err
			}
		} else {
			if err := g.updateRoot(); err != nil {
				return err
			}
		}
		if err := g.updateRouteProvider(); err != nil {
			return err
		}
		if !req.SkipWire {
			if err := g.generateWire(); err != nil {
				return err
			}
		}
		utils.Info("generate route success")
	case "biz":
		if err := g.GenBiz(); err != nil {
			return err
		}
		if err := g.updateBizProvider(); err != nil {
			return err
		}
		if req.WithIface {
			if err := g.GenBizIface(); err != nil {
				return err
			}
		}
		if !req.NoRepo {
			if err := g.GenRepo(); err != nil {
				return err
			}
			if err := g.updateRepoProvider(); err != nil {
				return err
			}
		}
		if req.WithParams {
			if err := g.GenParams(); err != nil {
				return err
			}
		}
		if req.WithContracts {
			if err := g.GenBizContracts(); err != nil {
				return err
			}
		}
		utils.Info("generate biz success")
	default:
		return errors.New("unknown type")
	}

	return nil
}

type fileSnapshot struct {
	content []byte
	exists  bool
}

func (g *Generator) snapshotMutationFiles() (func() error, error) {
	paths := []string{
		g.saveRouteFilePath, g.saveBizFilePath, g.saveBizIfaceFilePath, g.saveRepoFilePath,
		g.saveBizParamsFilePath, g.saveBizContractFilePath,
		filepath.Join(filepath.Dir(g.RootPath), "registry.go"),
		filepath.Join(filepath.Dir(g.RootPath), "root.go"),
		filepath.Join(g.RootPath, "provider.go"),
		filepath.Join(filepath.Dir(g.RootPath), "data", "provider.go"),
	}
	snapshots := make(map[string]fileSnapshot, len(paths))
	for _, path := range paths {
		if _, seen := snapshots[path]; seen {
			continue
		}
		content, readErr := os.ReadFile(path)
		if readErr == nil {
			snapshots[path] = fileSnapshot{content: content, exists: true}
			continue
		}
		if !errors.Is(readErr, os.ErrNotExist) {
			return nil, fmt.Errorf("snapshot %s: %w", path, readErr)
		}
		snapshots[path] = fileSnapshot{}
	}
	return func() error {
		var errs []error
		for path, snapshot := range snapshots {
			if snapshot.exists {
				if restoreErr := utils.WriteFileAtomic(path, snapshot.content, 0o644); restoreErr != nil {
					errs = append(errs, fmt.Errorf("restore %s: %w", path, restoreErr))
				}
				continue
			}
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				errs = append(errs, fmt.Errorf("remove %s: %w", path, removeErr))
			}
		}
		return errors.Join(errs...)
	}, nil
}

func (g *Generator) generateWire() error {
	projectRoot, err := utils.FindProjectRoot(8)
	if err != nil {
		return fmt.Errorf("find project root for Wire: %w", err)
	}
	cmd := exec.Command("wire", "./cmd/app")
	cmd.Dir = projectRoot
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("regenerate Wire: %w", err)
	}
	return nil
}

func (g *Generator) prepare() error {
	g.IsSQLC = g.detectProjectKind().IsSQLC()

	if g.WithCRUD {
		if g.IsSQLC {
			if g.Model == "" {
				g.Model = g.EntName
			}
			if g.Model == "" {
				return errors.New("please specify --model (e.g. --model User) for a sqlc project")
			}
			g.Model = strcase.ToCamel(g.Model)
			if g.Plural == "" {
				g.Plural = g.Model + "s"
			}
			g.Plural = strcase.ToCamel(g.Plural)
			if g.Table == "" {
				g.Table = strcase.ToSnake(g.Plural)
			}
			g.warnMissingSQLCModel()
		} else if g.EntName == "" {
			return errors.New("please specify ent name")
		}
	}

	g.FilenameRouteSuffix = "_routes.go"
	g.FilenameBizSuffix = "_biz.go"
	g.FilenameRepoSuffix = "_repo.go"
	g.RouteNameSuffix = "Routes"

	goPackage := os.Getenv("GOPACKAGE")
	if goPackage != "" {
		// Invoked via //go:generate — use the environment-provided context.
		g.PackageName = goPackage
		g.RootPath = utils.CurrentDir()
	} else {
		// Invoked from the command line — walk up/down to find the project root.
		root, err := utils.FindProjectRoot(8)
		if err != nil {
			return fmt.Errorf("cannot find project root (go.mod): %w", err)
		}
		switch g.TypeName {
		case "route":
			pkg := g.Pkg
			if pkg == "" {
				return errors.New("please specify --pkg (e.g. --pkg v1) when running outside //go:generate")
			}
			g.PackageName = pkg
			g.RootPath = filepath.Join(root, "internal", "routes", pkg)
		case "biz":
			g.PackageName = "biz"
			g.RootPath = filepath.Join(root, "internal", "biz")
		}
	}

	g.ModName = utils.GetModName()

	g.saveRouteFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+g.FilenameRouteSuffix)
	g.saveBizFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+g.FilenameBizSuffix)
	g.saveBizIfaceFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+"_biz_iface.go")
	g.saveRepoFilePath = filepath.Join(filepath.Dir(g.RootPath), "data", strcase.ToSnake(g.Name)+g.FilenameRepoSuffix)
	g.saveBizParamsFilePath = filepath.Join(g.RootPath, "../domain/params", strcase.ToSnake(g.Name)+"_params.go")
	g.saveBizContractFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+"_contract.go")

	g.routeTpl = tpl.RouteTpl
	g.bizParamsTpl = tpl.BizParamsTpl
	g.bizContractTpl = tpl.BizContractTpl

	if g.IsSQLC && g.WithCRUD {
		g.bizTpl = tpl.BizSQLCTpl
		g.bizIfaceTpl = tpl.BizIfaceSQLCTpl
		g.repoTpl = tpl.RepoSQLCTpl
	} else {
		g.bizTpl = tpl.BizTpl
		g.bizIfaceTpl = tpl.BizIfaceTpl
		g.repoTpl = tpl.RepoTpl
	}

	g.StructRouteName = strcase.ToCamel(g.Name + "Route")
	g.StructBizName = strcase.ToCamel(g.Name + "Biz")
	g.StructRepoName = strcase.ToCamel(g.Name + "Repository")
	g.V2 = g.V2 || g.usesKoanfV2()
	if !g.V2 && g.usesLegacyKoanf() {
		utils.Info("warning: legacy github.com/knadh/koanf detected; run `nr migrate koanf` to preview the v2 migration")
	}
	g.UseRouteRegistry = g.hasRouteRegistry()
	g.UseTypedAPI = g.hasTypedAPI()
	g.UseErrorTypedAPI = g.hasErrorTypedAPI()
	g.UseRouterRegister = g.hasRouterRegister()

	return nil
}

// warnMissingSQLCModel prints a hint when the requested sqlc model type has
// not been generated yet. The scaffold still compiles only after the table and
// queries exist, so surfacing the next step avoids a confusing build error.
func (g *Generator) warnMissingSQLCModel() {
	root, err := utils.FindProjectRoot(8)
	if err != nil {
		return
	}
	models, err := os.ReadFile(filepath.Join(root, "internal", "data", "sqlc", "models.go"))
	if err != nil {
		return
	}
	if !strings.Contains(string(models), "type "+g.Model+" struct") {
		utils.Info(fmt.Sprintf("warning: sqlc.%s is not generated yet; add the %s table to db/migrations and run make sqlc before building", g.Model, g.Table))
	}
}

// detectProjectKind classifies the project containing the current directory so
// the generator can emit Ent or sqlc persistence code.
func (g *Generator) detectProjectKind() core.ProjectKind {
	root, err := utils.FindProjectRoot(8)
	if err != nil {
		return core.ProjectKindUnknown
	}
	return core.DetectProjectKind(root)
}

// usesKoanfV2 detects the project's configured koanf major version so a
// scaffold generated from the current template compiles without requiring an
// otherwise easy-to-miss --v2 flag.
func (g *Generator) usesKoanfV2() bool {
	root, err := utils.FindProjectRoot(8)
	if err != nil {
		return false
	}
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(goMod), "github.com/knadh/koanf/v2")
}

func (g *Generator) usesLegacyKoanf() bool {
	root, err := utils.FindProjectRoot(8)
	if err != nil {
		return false
	}
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(goMod), "github.com/knadh/koanf ")
}

func (g *Generator) hasTypedAPI() bool {
	root := filepath.Dir(filepath.Dir(g.RootPath))
	source, err := os.ReadFile(filepath.Join(root, "core", "rtn.go"))
	if err != nil {
		return false
	}
	return strings.Contains(string(source), "func NoInput[")
}

func (g *Generator) hasErrorTypedAPI() bool {
	root := filepath.Dir(filepath.Dir(g.RootPath))
	source, err := os.ReadFile(filepath.Join(root, "core", "rtn.go"))
	if err != nil {
		return false
	}
	return strings.Contains(string(source), "func NoInputE[")
}

func (g *Generator) hasRouteRegistry() bool {
	registryPath := filepath.Join(filepath.Dir(g.RootPath), "registry.go")
	source, err := os.ReadFile(registryPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(source), "func NewRouteRegistry(")
}

func (g *Generator) hasRouterRegister() bool {
	registryPath := filepath.Join(filepath.Dir(g.RootPath), "registry.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, registryPath, nil, 0)
	if err != nil {
		return false
	}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Registrar" {
				continue
			}
			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok || iface.Methods == nil {
				continue
			}
			for _, method := range iface.Methods.List {
				if len(method.Names) != 1 || method.Names[0].Name != "Register" {
					continue
				}
				fn, ok := method.Type.(*ast.FuncType)
				if !ok || fn.Params == nil || len(fn.Params.List) != 1 {
					continue
				}
				selector, ok := fn.Params.List[0].Type.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "IRouter" {
					continue
				}
				if pkg, ok := selector.X.(*ast.Ident); ok && pkg.Name == "gin" {
					return true
				}
			}
		}
	}
	return false
}

func (g *Generator) GenRoute() error {
	return g.renderTemplateToFile("route", g.routeTpl, g.saveRouteFilePath)
}

func (g *Generator) GenBiz() error {
	return g.renderTemplateToFile("biz", g.bizTpl, g.saveBizFilePath)
}

func (g *Generator) GenBizIface() error {
	if err := g.renderTemplateToFile("biz_iface", g.bizIfaceTpl, g.saveBizIfaceFilePath); err != nil {
		return err
	}
	return g.appendMockGenDirective()
}

// appendMockGenDirective adds a //go:generate mockgen line to mocks/mock_gen.go.
func (g *Generator) appendMockGenDirective() error {
	mockGenPath := filepath.Join(g.RootPath, "mocks", "mock_gen.go")
	if !utils.CheckExist(mockGenPath) {
		utils.Info("mocks/mock_gen.go not found, skipping mockgen directive")
		return nil
	}

	snakeName := strcase.ToSnake(g.Name)
	line := fmt.Sprintf(
		"//go:generate mockgen -source=../%s_biz_iface.go -destination=mock_%s_biz.go -package=mocks\n",
		snakeName, snakeName,
	)

	content, err := os.ReadFile(mockGenPath)
	if err != nil {
		return fmt.Errorf("read mocks/mock_gen.go: %w", err)
	}

	if strings.Contains(string(content), line) {
		utils.Info("mockgen directive already present, skipping")
		return nil
	}

	// Insert the directive line before "package mocks" so the file stays syntactically valid.
	updated := strings.Replace(string(content), "package mocks", line+"package mocks", 1)
	if err := os.WriteFile(mockGenPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("write mocks/mock_gen.go: %w", err)
	}

	utils.Info("added mockgen directive to mocks/mock_gen.go")
	return nil
}

func (g *Generator) GenParams() error {
	if err := g.renderTemplateToFile("params", g.bizParamsTpl, g.saveBizParamsFilePath); err != nil {
		return err
	}

	utils.Info("generate params success")
	return nil
}

func (g *Generator) GenBizContracts() error {
	if err := g.renderTemplateToFile("biz_contract", g.bizContractTpl, g.saveBizContractFilePath); err != nil {
		return err
	}

	utils.Info("generate biz contracts success")
	return nil
}

func (g *Generator) GenRepo() error {
	return g.renderTemplateToFile("repo", g.repoTpl, g.saveRepoFilePath)
}

func (g *Generator) renderTemplateToFile(name string, tplText string, savePath string) error {
	if err := g.checkFile(savePath); err != nil {
		return err
	}

	parsed, err := template.New(name).Parse(tplText)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	if err := parsed.Execute(buffer, g); err != nil {
		return err
	}

	source, err := format.Source(buffer.Bytes())
	if err != nil {
		return err
	}

	return utils.SaveToFile(savePath, source, false)
}

func (g *Generator) updateRoot() error {
	utils.Info("updating root.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "root.go")
	if !utils.CheckExist(rootFilePath) {
		return fmt.Errorf("can't find root.go file [%s]", rootFilePath)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	qualifier := g.addPackageImport(fset, f)
	walker := visitor.NewUpdateRoot(fmt.Sprintf("%s%s", qualifier, g.StructRouteName), fmt.Sprintf("*%s.%s", qualifier, g.StructRouteName))
	ast.Walk(walker, f)

	var dst bytes.Buffer
	if err := format.Node(&dst, fset, f); err != nil {
		return err
	}

	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatHttpEngine(dst.Bytes())
	if err != nil {
		return err
	}

	if err := utils.SaveToFile(rootFilePath, rtn, true); err != nil {
		return err
	}

	utils.Info("updating root.go success")
	return nil
}

// updateRouteRegistry adds a newly generated route to the new template's
// central registry.  It deliberately does not touch root.go: projects without
// this opt-in structure continue through updateRoot above.
func (g *Generator) updateRouteRegistry() error {
	registryPath := filepath.Join(filepath.Dir(g.RootPath), "registry.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, registryPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse route registry %s: %w", registryPath, err)
	}

	var registry *ast.FuncDecl
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && fd.Name.Name == "NewRouteRegistry" {
			registry = fd
			break
		}
	}
	if registry == nil || registry.Type.Params == nil || registry.Body == nil {
		return fmt.Errorf("can't find NewRouteRegistry in %s", registryPath)
	}

	varName := strcase.ToLowerCamel(g.StructRouteName)
	for _, field := range registry.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == varName {
				return fmt.Errorf("route registry already contains %s", varName)
			}
		}
	}
	qualifier := g.addPackageImport(fset, f)
	typeExpr, err := parser.ParseExpr(fmt.Sprintf("*%s.%s", qualifier, g.StructRouteName))
	if err != nil {
		return err
	}
	registry.Type.Params.List = append(registry.Type.Params.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(varName)},
		Type:  typeExpr,
	})

	updatedReturn := false
	for _, stmt := range registry.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		if literal, ok := ret.Results[0].(*ast.CompositeLit); ok {
			literal.Elts = append(literal.Elts, ast.NewIdent(varName))
			updatedReturn = true
			break
		}
	}
	if !updatedReturn {
		return fmt.Errorf("NewRouteRegistry must return a RouteRegistry composite literal")
	}

	var dst bytes.Buffer
	if err := format.Node(&dst, fset, f); err != nil {
		return err
	}
	if err := utils.SaveToFile(registryPath, dst.Bytes(), true); err != nil {
		return err
	}
	utils.Info("updating route registry success")
	return nil
}

func (g *Generator) updateBizProvider() error {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(g.RootPath, "provider.go")
	if !utils.CheckExist(rootFilePath) {
		return fmt.Errorf("can't find provider.go file [%s]", rootFilePath)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	visitor.UpdateProvider(f, "ProviderBizSet", fmt.Sprintf("New%s", g.StructBizName))

	var dst bytes.Buffer
	if err := format.Node(&dst, fset, f); err != nil {
		return err
	}

	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatProvider(dst.Bytes())
	if err != nil {
		return err
	}

	if err := utils.SaveToFile(rootFilePath, rtn, true); err != nil {
		return err
	}

	utils.Info("updating provider.go success")
	return nil
}

func (g *Generator) updateRepoProvider() error {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "data", "provider.go")
	if !utils.CheckExist(rootFilePath) {
		return fmt.Errorf("can't find provider.go file [%s]", rootFilePath)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	visitor.UpdateProvider(f, "ProviderDataSet", fmt.Sprintf("New%s", g.StructRepoName))

	var dst bytes.Buffer
	if err := format.Node(&dst, fset, f); err != nil {
		return err
	}

	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatProvider(dst.Bytes())
	if err != nil {
		return err
	}

	if err := utils.SaveToFile(rootFilePath, rtn, true); err != nil {
		return err
	}

	utils.Info("updating provider.go success")
	return nil
}

func (g *Generator) updateRouteProvider() error {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "provider.go")
	if !utils.CheckExist(rootFilePath) {
		return fmt.Errorf("can't find provider.go file [%s]", rootFilePath)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	qualifier := g.addPackageImport(fset, f)
	walker := visitor.NewRouteProvideVisitor(qualifier, fmt.Sprintf("New%s", g.StructRouteName))
	ast.Walk(walker, f)
	if !walker.Updated() {
		return fmt.Errorf("can't find ProviderRouteSet wire.NewSet declaration in %s", rootFilePath)
	}

	var dst bytes.Buffer
	if err := format.Node(&dst, fset, f); err != nil {
		return err
	}

	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatProvider(dst.Bytes())
	if err != nil {
		return err
	}

	if err := utils.SaveToFile(rootFilePath, rtn, true); err != nil {
		return err
	}

	utils.Info("updating provider.go success")
	return nil
}

// addPackageImport imports the generated route package with its natural Go
// package name whenever possible. An explicit alias is used only to avoid a
// collision with an existing import; this keeps registry imports idiomatic
// without relying on directory-name heuristics such as a "v" prefix.
func (g *Generator) addPackageImport(fset *token.FileSet, f *ast.File) string {
	path := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, "\"") != path {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return g.PackageName
	}

	used := map[string]bool{}
	for _, imp := range f.Imports {
		name := filepath.Base(strings.Trim(imp.Path.Value, "\""))
		if imp.Name != nil {
			name = imp.Name.Name
		}
		used[name] = true
	}
	qualifier := g.PackageName
	if used[qualifier] {
		base := qualifier + "route"
		qualifier = base
		for i := 2; used[qualifier]; i++ {
			qualifier = base + strconv.Itoa(i)
		}
		_ = astutil.AddNamedImport(fset, f, qualifier, path)
		return qualifier
	}
	_ = astutil.AddImport(fset, f, path)
	return qualifier
}

func (g *Generator) checkFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	return nil
}

func (g *Generator) ToLowerCamel(str string) string {
	return strcase.ToLowerCamel(str)
}

func (g *Generator) ToCamel(str string) string {
	return strcase.ToCamel(str)
}

func (g *Generator) ToSnake(str string) string {
	return strcase.ToSnake(str)
}

func (g *Generator) ToKebab(str string) string {
	return strcase.ToKebab(str)
}

func (g *Generator) ExtractInitials(str string) string {
	return utils.ExtractInitials(g.ToCamel(str))
}
