/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

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

	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/Xwudao/neter-template/pkg/config"
	"github.com/Xwudao/neter/internal/tpl"
	"github.com/Xwudao/neter/internal/visitor"
	"github.com/Xwudao/neter/pkg/utils"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "gen some for neter",
	Long:  `gen route, service, or other things for neter`,
	Run: func(cmd *cobra.Command, args []string) {
		tpe, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")

		g := NewGenerate(name, tpe)
		g.init()

		switch tpe {
		case "route":
			g.GenRoute()
			g.updateRoot()
			g.updateRouteProvider()
			utils.Info("generate route success")
		case "biz":
			g.GenBiz()
			g.updateBizProvider()
			g.GenRepo()
			g.updateRepoProvider()
			utils.Info("generate biz success")
		default:
			utils.CheckErrWithStatus(errors.New("unknown type"))
		}

	},
}

type GenerateRoute struct {
	RootPath        string
	RouteNameSuffix string
	PackageName     string
	ModName         string

	routeTpl string
	bizTpl   string
	repoTpl  string

	FilenameRouteSuffix string
	FilenameBizSuffix   string
	FilenameRepoSuffix  string

	saveRouteFilePath string //eg: home_route.go
	saveBizFilePath   string //eg: home_biz.go
	saveRepoFilePath  string //eg: home_repo.go

	Name     string //eg: home
	TypeName string //eg: route, service, etc

	StructRouteName string //eg: HomeRoute
	StructBizName   string //eg: HomeBiz
	StructRepoName  string //eg: HomeRepo
}

func NewGenerate(name string, typeName string) *GenerateRoute {
	return &GenerateRoute{Name: name, TypeName: typeName}
}

func (g *GenerateRoute) init() {
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

	g.routeTpl = tpl.RouteTpl
	g.bizTpl = tpl.BizTpl
	g.repoTpl = tpl.RepoTpl

	g.StructRouteName = strcase.ToCamel(g.Name + "Route")
	g.StructBizName = strcase.ToCamel(g.Name + "Biz")
	g.StructRepoName = strcase.ToCamel(g.Name + "Repository")

	if g.PackageName == "" {
		utils.CheckErrWithStatus(errors.New("please run with //go:generate"))
		return
	}
}

func (g *GenerateRoute) GenRoute() {
	g.checkFile(g.saveRouteFilePath)

	parse, err := template.New("route").Parse(g.routeTpl)
	utils.CheckErrWithStatus(err)

	buffer := bytes.NewBuffer([]byte{})
	err = parse.Execute(buffer, g)
	utils.CheckErrWithStatus(err)

	source, err := format.Source(buffer.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(g.saveRouteFilePath, source, false)
	utils.CheckErrWithStatus(err)

}

func (g *GenerateRoute) GenBiz() {
	g.checkFile(g.saveBizFilePath)

	parse, err := template.New("biz").Parse(g.bizTpl)
	utils.CheckErrWithStatus(err)

	buffer := bytes.NewBuffer([]byte{})
	err = parse.Execute(buffer, g)
	utils.CheckErrWithStatus(err)

	source, err := format.Source(buffer.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(g.saveBizFilePath, source, false)
	utils.CheckErrWithStatus(err)
}

func (g *GenerateRoute) GenRepo() {
	g.checkFile(g.saveRepoFilePath)

	parse, err := template.New("repo").Parse(g.repoTpl)
	utils.CheckErrWithStatus(err)

	buffer := bytes.NewBuffer([]byte{})
	err = parse.Execute(buffer, g)
	utils.CheckErrWithStatus(err)

	source, err := format.Source(buffer.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(g.saveRepoFilePath, source, false)
	utils.CheckErrWithStatus(err)
}

func (g *GenerateRoute) updateRoot() {
	utils.Info("updating root.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "root.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find root.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	//update content
	walker := visitor.NewUpdateRoot(fmt.Sprintf("%s%s", g.PackageName, g.StructRouteName), fmt.Sprintf("*%s.%s", g.PackageName, g.StructRouteName))
	ast.Walk(walker, f)

	//update imports
	pkgName := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	_ = astutil.AddNamedImport(fset, f, g.PackageName, pkgName)
	//if !added {
	//	utils.CheckErrWithStatus(fmt.Errorf("can't add import [%s]", pkgName))
	//}

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	err = utils.SaveToFile(rootFilePath, dst.Bytes(), true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating root.go success")
}

//update biz provider
func (g *GenerateRoute) updateBizProvider() {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(g.RootPath, "provider.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find provider.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	//update content
	walker := visitor.NewCurrentProvideVisitor(fmt.Sprintf("New%s", g.StructBizName))
	ast.Walk(walker, f)

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	err = utils.SaveToFile(rootFilePath, dst.Bytes(), true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating provider.go success")
}

//update repo provider
func (g *GenerateRoute) updateRepoProvider() {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "data", "provider.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find provider.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	//update content
	walker := visitor.NewCurrentProvideVisitor(fmt.Sprintf("New%s", g.StructRepoName))
	ast.Walk(walker, f)

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	err = utils.SaveToFile(rootFilePath, dst.Bytes(), true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating provider.go success")
}

//update route provider
func (g *GenerateRoute) updateRouteProvider() {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "provider.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find provider.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	//update content
	walker := visitor.NewRouteProvideVisitor(g.PackageName, fmt.Sprintf("New%s", g.StructRouteName))
	ast.Walk(walker, f)

	//update imports
	pkgName := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	_ = astutil.AddNamedImport(fset, f, g.PackageName, pkgName)
	//if !added {
	//	utils.CheckErrWithStatus(fmt.Errorf("can't add import [%s]", pkgName))
	//}

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	err = utils.SaveToFile(rootFilePath, dst.Bytes(), true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating provider.go success")
}

func (g *GenerateRoute) checkFile(p string) {
	if _, err := os.Stat(p); err == nil {
		utils.CheckErrWithStatus(errors.New("file already exists"))
		return
	}
}

//template functions

func (g *GenerateRoute) ToLowerCamel(str string) string {
	return strcase.ToLowerCamel(str)
}

func (g *GenerateRoute) ToCamel(str string) string {
	return strcase.ToCamel(str)
}

// sub commands

var genEntCmd = &cobra.Command{
	Use:   "ent",
	Short: "generate entity",
	Long:  `generate entity`,
	Run: func(cmd *cobra.Command, args []string) {
		koanf, err := config.NewConfig()
		utils.CheckErrWithStatus(err)

		prefix, _ := cmd.Flags().GetString("prefix")
		if prefix == "" {
			prefix = koanf.String("db.tablePrefix")
		}

		cwd, _ := os.Getwd()
		schemaPath := filepath.Join(cwd, "internal/data//ent/schema")
		err = entc.Generate(schemaPath, &gen.Config{
			Hooks: []gen.Hook{
				PrefixSchema(prefix),
			},
			IDType:   &field.TypeInfo{Type: field.TypeInt64},
			Features: []gen.Feature{gen.FeatureVersionedMigration, gen.FeatureModifier},
		})
		utils.CheckErrWithStatus(err)

		utils.Info("generate entity success")
	},
}

// PrefixSchema add the prefix to the schema name
func PrefixSchema(prefix string) gen.Hook {
	return func(next gen.Generator) gen.Generator {
		return gen.GenerateFunc(func(g *gen.Graph) error {
			if prefix == "" {
				return next.Generate(g)
			}
			if !strings.HasSuffix(prefix, "_") {
				prefix += "_"
			}
			for _, n := range g.Nodes {
				a := &entsql.Annotation{Table: fmt.Sprintf("%s%s", strcase.ToSnake(prefix), n.Table())}
				if n.Annotations == nil {
					n.Annotations = gen.Annotations{}
				}
				n.Annotations[a.Name()] = a
			}
			return next.Generate(g)
		})
	}
}

func init() {
	genCmd.AddCommand(genEntCmd)
	rootCmd.AddCommand(genCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// genCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// genCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	genCmd.Flags().StringP("type", "t", "route", "type of gen")
	genCmd.Flags().StringP("name", "n", "", "name of gen")

	genCmd.MarkFlagsRequiredTogether("type", "name")

	genEntCmd.Flags().StringP("prefix", "p", "", "prefix of entity")

}
