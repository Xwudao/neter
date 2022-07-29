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
			g.updateProvider()
			utils.Info("generate route success")

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
	FilenameSuffix  string
	RouteNameSuffix string
	PackageName     string
	ModName         string

	routeTpl string

	saveFilePath string //eg: home_route.go

	Name       string //eg: home
	TypeName   string //eg: route, service, etc
	StructName string //eg: HomeRoute
}

func NewGenerate(name string, typeName string) *GenerateRoute {
	return &GenerateRoute{Name: name, TypeName: typeName}
}

func (g *GenerateRoute) init() {
	g.FilenameSuffix = "_routes.go"
	g.RouteNameSuffix = "Routes"
	g.PackageName = os.Getenv("GOPACKAGE")
	g.ModName = utils.GetModName()
	g.RootPath = utils.CurrentDir()
	g.saveFilePath = filepath.Join(g.RootPath, strcase.ToSnake(g.Name)+g.FilenameSuffix)

	g.routeTpl = tpl.RouteTpl

	g.StructName = strcase.ToCamel(g.Name + "Route")

	if g.PackageName == "" {
		utils.CheckErrWithStatus(errors.New("please run with //go:generate"))
		return
	}
}

func (g *GenerateRoute) GenRoute() {
	g.checkFile()

	parse, err := template.New("route").Parse(g.routeTpl)
	utils.CheckErrWithStatus(err)

	buffer := bytes.NewBuffer([]byte{})
	err = parse.Execute(buffer, g)
	utils.CheckErrWithStatus(err)

	source, err := format.Source(buffer.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(g.saveFilePath, source, false)
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
	walker := visitor.NewUpdateRoot(fmt.Sprintf("%s%s", g.PackageName, g.StructName), fmt.Sprintf("*%s.%s", g.PackageName, g.StructName))
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

//update provider
func (g *GenerateRoute) updateProvider() {
	utils.Info("updating injector.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "injector.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find injector.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	//update content
	walker := visitor.NewProvideVisitor(g.PackageName, fmt.Sprintf("New%s", g.StructName))
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

	utils.Info("updating injector.go success")
}

func (g *GenerateRoute) checkFile() {
	if _, err := os.Stat(g.saveFilePath); err == nil {
		utils.CheckErrWithStatus(errors.New("file already exists"))
		return
	}
}
