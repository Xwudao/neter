/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"errors"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/tpl"
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

func (g *GenerateRoute) checkFile() {
	if _, err := os.Stat(g.saveFilePath); err == nil {
		utils.CheckErrWithStatus(errors.New("file already exists"))
		return
	}
}
