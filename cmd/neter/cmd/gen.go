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
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/ast/astutil"

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
			utils.Info("generate biz success")
		default:
			utils.CheckErrWithStatus(errors.New("unknown type"))
		}

	},
}

func init() {
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
}

type GenerateRoute struct {
	RootPath        string
	RouteNameSuffix string
	PackageName     string
	ModName         string

	routeTpl string
	bizTpl   string

	FilenameRouteSuffix string
	FilenameBizSuffix   string

	saveRouteFilePath string //eg: home_route.go
	saveBizFilePath   string //eg: home_biz.go

	Name     string //eg: home
	TypeName string //eg: route, service, etc

	StructRouteName string //eg: HomeRoute
	StructBizName   string //eg: HomeBiz
}

func NewGenerate(name string, typeName string) *GenerateRoute {
	return &GenerateRoute{Name: name, TypeName: typeName}
}

func (g *GenerateRoute) init() {
	g.FilenameRouteSuffix = "_routes.go"
	g.FilenameBizSuffix = "_biz.go"
	g.RouteNameSuffix = "Routes"
	g.PackageName = os.Getenv("GOPACKAGE")
	g.ModName = utils.GetModName()
	g.RootPath = utils.CurrentDir()
	g.saveRouteFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+g.FilenameRouteSuffix)
	g.saveBizFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+g.FilenameBizSuffix)

	g.routeTpl = tpl.RouteTpl
	g.bizTpl = tpl.BizTpl

	g.StructRouteName = strcase.ToCamel(g.Name + "Route")
	g.StructBizName = strcase.ToCamel(g.Name + "Biz")

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
	walker := visitor.NewBizProvideVisitor(fmt.Sprintf("New%s", g.StructBizName))
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
