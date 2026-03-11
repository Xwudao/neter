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
	"path/filepath"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/Xwudao/neter/internal/tpl"
	"github.com/Xwudao/neter/internal/visitor"
	"github.com/Xwudao/neter/pkg/utils"
)

type Generator struct {
	RootPath        string
	RouteNameSuffix string
	PackageName     string
	ModName         string

	WithCRUD bool
	EntName  string
	V2       bool

	routeTpl     string
	bizTpl       string
	repoTpl      string
	bizParamsTpl string

	FilenameRouteSuffix string
	FilenameBizSuffix   string
	FilenameRepoSuffix  string

	saveRouteFilePath     string
	saveBizFilePath       string
	saveRepoFilePath      string
	saveBizParamsFilePath string

	Name     string
	TypeName string

	StructRouteName string
	StructBizName   string
	StructRepoName  string
}

type Request struct {
	TypeName   string
	Name       string
	NoRepo     bool
	WithCRUD   bool
	WithParams bool
	EntName    string
	V2         bool
}

func NewGenerator(req Request) *Generator {
	return &Generator{
		Name:     req.Name,
		TypeName: req.TypeName,
		WithCRUD: req.WithCRUD,
		EntName:  req.EntName,
		V2:       req.V2,
	}
}

func Execute(req Request) error {
	g := NewGenerator(req)
	if err := g.prepare(); err != nil {
		return err
	}

	switch req.TypeName {
	case "route":
		if err := g.GenRoute(); err != nil {
			return err
		}
		if err := g.updateRoot(); err != nil {
			return err
		}
		if err := g.updateRouteProvider(); err != nil {
			return err
		}
		utils.Info("generate route success")
	case "biz":
		if err := g.GenBiz(); err != nil {
			return err
		}
		if err := g.updateBizProvider(); err != nil {
			return err
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
		utils.Info("generate biz success")
	default:
		return errors.New("unknown type")
	}

	return nil
}

func (g *Generator) prepare() error {
	if g.WithCRUD && g.EntName == "" {
		return errors.New("please specify ent name")
	}

	g.FilenameRouteSuffix = "_routes.go"
	g.FilenameBizSuffix = "_biz.go"
	g.FilenameRepoSuffix = "_repo.go"
	g.RouteNameSuffix = "Routes"
	g.PackageName = os.Getenv("GOPACKAGE")
	g.ModName = utils.GetModName()
	g.RootPath = utils.CurrentDir()

	g.saveRouteFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+g.FilenameRouteSuffix)
	g.saveBizFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+g.FilenameBizSuffix)
	g.saveRepoFilePath = filepath.Join(filepath.Dir(g.RootPath), "data", strcase.ToSnake(g.Name)+g.FilenameRepoSuffix)
	g.saveBizParamsFilePath = filepath.Join(g.RootPath, "../domain/params", strcase.ToSnake(g.Name)+"_params.go")

	g.routeTpl = tpl.RouteTpl
	g.bizTpl = tpl.BizTpl
	g.repoTpl = tpl.RepoTpl
	g.bizParamsTpl = tpl.BizParamsTpl

	g.StructRouteName = strcase.ToCamel(g.Name + "Route")
	g.StructBizName = strcase.ToCamel(g.Name + "Biz")
	g.StructRepoName = strcase.ToCamel(g.Name + "Repository")

	if g.PackageName == "" {
		return errors.New("please run with //go:generate")
	}

	return nil
}

func (g *Generator) GenRoute() error {
	return g.renderTemplateToFile("route", g.routeTpl, g.saveRouteFilePath)
}

func (g *Generator) GenBiz() error {
	return g.renderTemplateToFile("biz", g.bizTpl, g.saveBizFilePath)
}

func (g *Generator) GenParams() error {
	if err := g.renderTemplateToFile("params", g.bizParamsTpl, g.saveBizParamsFilePath); err != nil {
		return err
	}

	utils.Info("generate params success")
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

	walker := visitor.NewUpdateRoot(fmt.Sprintf("%s%s", g.PackageName, g.StructRouteName), fmt.Sprintf("*%s.%s", g.PackageName, g.StructRouteName))
	ast.Walk(walker, f)

	g.addPackageImport(fset, f)

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

	walker := visitor.NewRouteProvideVisitor(g.PackageName, fmt.Sprintf("New%s", g.StructRouteName))
	ast.Walk(walker, f)

	g.addPackageImport(fset, f)

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

func (g *Generator) addPackageImport(fset *token.FileSet, f *ast.File) {
	pkgName := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	if strings.HasPrefix(g.PackageName, "v") {
		_ = astutil.AddNamedImport(fset, f, g.PackageName, pkgName)
		return
	}

	_ = astutil.AddImport(fset, f, pkgName)
}

func (g *Generator) checkFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return errors.New("file already exists")
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
