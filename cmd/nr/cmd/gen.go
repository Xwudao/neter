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
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/filex"
	"github.com/Xwudao/neter/pkg/typex"

	"github.com/Xwudao/neter/internal/tpl"
	"github.com/Xwudao/neter/internal/visitor"
	"github.com/Xwudao/neter/pkg/utils"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "gen some for nr",
	Long:  `gen route, service, or other things for nr`,
	Run: func(cmd *cobra.Command, args []string) {
		tpe, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")
		noRepo, _ := cmd.Flags().GetBool("no-repo")
		v2, _ := cmd.Flags().GetBool("v2")
		withCrud, _ := cmd.Flags().GetBool("with-crud")
		entName, _ := cmd.Flags().GetString("ent-name")

		g := NewGenerate(name, tpe, withCrud, entName, v2)

		err := g.preCheck()
		utils.CheckErrWithStatus(err)

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
			if !noRepo {
				g.GenRepo()
				g.updateRepoProvider()
			}
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

	WithCRUD bool
	EntName  string

	routeTpl string
	bizTpl   string
	repoTpl  string

	FilenameRouteSuffix string
	FilenameBizSuffix   string
	FilenameRepoSuffix  string

	saveRouteFilePath string // eg: home_route.go
	saveBizFilePath   string // eg: home_biz.go
	saveRepoFilePath  string // eg: home_repo.go

	Name     string // eg: home
	TypeName string // eg: route, service, etc
	NoRepo   bool   // no repo file generated

	V2 bool // is v2, now just changed the v2 koanf

	StructRouteName string // eg: HomeRoute
	StructBizName   string // eg: HomeBiz
	StructRepoName  string // eg: HomeRepo
}

func NewGenerate(name string, typeName string, crud bool, entName string, v2 bool) *GenerateRoute {
	return &GenerateRoute{Name: name, TypeName: typeName, WithCRUD: crud, EntName: entName, V2: v2}
}

func (g *GenerateRoute) preCheck() error {
	if g.WithCRUD && g.EntName == "" {
		return errors.New("please specify ent name")
	}

	return nil
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
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	utils.CheckErrWithStatus(err)

	// update content
	walker := visitor.NewUpdateRoot(fmt.Sprintf("%s%s", g.PackageName, g.StructRouteName), fmt.Sprintf("*%s.%s", g.PackageName, g.StructRouteName))
	ast.Walk(walker, f)

	// update imports
	pkgName := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	if strings.HasPrefix(g.PackageName, "v") {
		_ = astutil.AddNamedImport(fset, f, g.PackageName, pkgName)
	} else {
		_ = astutil.AddImport(fset, f, pkgName)
	}
	// if !added {
	//	utils.CheckErrWithStatus(fmt.Errorf("can't add import [%s]", pkgName))
	// }

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)

	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatHttpEngine(dst.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(rootFilePath, rtn, true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating root.go success")
}

// update biz provider
func (g *GenerateRoute) updateBizProvider() {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(g.RootPath, "provider.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find provider.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	utils.CheckErrWithStatus(err)

	// update content
	//walker := visitor.NewCurrentProvideVisitor(fmt.Sprintf("New%s", g.StructBizName))
	//ast.Walk(walker, f)
	visitor.UpdateProvider(f, "ProviderBizSet", fmt.Sprintf("New%s", g.StructBizName))

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)

	// reformat
	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatProvider(dst.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(rootFilePath, rtn, true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating provider.go success")
}

// update repo provider
func (g *GenerateRoute) updateRepoProvider() {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "data", "provider.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find provider.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	utils.CheckErrWithStatus(err)

	// update content
	//walker := visitor.NewCurrentProvideVisitor(fmt.Sprintf("New%s", g.StructRepoName))
	//ast.Walk(walker, f)
	visitor.UpdateProvider(f, "ProviderDataSet", fmt.Sprintf("New%s", g.StructRepoName))

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)

	// reformat
	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatProvider(dst.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(rootFilePath, rtn, true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating provider.go success")
}

// update route provider
func (g *GenerateRoute) updateRouteProvider() {
	utils.Info("updating provider.go")
	rootFilePath := filepath.Join(filepath.Dir(g.RootPath), "provider.go")
	exist := utils.CheckExist(rootFilePath)
	if !exist {
		utils.CheckErrWithStatus(fmt.Errorf("can't find provider.go file [%s]", rootFilePath))
	}

	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, rootFilePath, nil, parser.ParseComments)
	utils.CheckErrWithStatus(err)

	// update content
	walker := visitor.NewRouteProvideVisitor(g.PackageName, fmt.Sprintf("New%s", g.StructRouteName))
	ast.Walk(walker, f)

	// update imports
	pkgName := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	if strings.HasPrefix(g.PackageName, "v") {
		_ = astutil.AddNamedImport(fset, f, g.PackageName, pkgName)
	} else {
		_ = astutil.AddImport(fset, f, pkgName)
	}
	// if !added {
	//	utils.CheckErrWithStatus(fmt.Errorf("can't add import [%s]", pkgName))
	// }

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)

	// reformat
	formatter := visitor.NewFormatLine()
	rtn, err := formatter.FormatProvider(dst.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(rootFilePath, rtn, true)
	utils.CheckErrWithStatus(err)

	utils.Info("updating provider.go success")
}

func (g *GenerateRoute) checkFile(p string) {
	if _, err := os.Stat(p); err == nil {
		utils.CheckErrWithStatus(errors.New("file already exists"))
		return
	}
}

// template functions

func (g *GenerateRoute) ToLowerCamel(str string) string {
	return strcase.ToLowerCamel(str)
}

func (g *GenerateRoute) ToCamel(str string) string {
	return strcase.ToCamel(str)
}

func (g *GenerateRoute) ToSnake(str string) string {
	return strcase.ToSnake(str)
}

func (g *GenerateRoute) ToKebab(str string) string {
	return strcase.ToKebab(str)
}
func (g *GenerateRoute) ExtractInitials(str string) string {
	return utils.ExtractInitials(g.ToCamel(str))
}

// sub commands
var genEntCmd = &cobra.Command{
	Use:   "ent",
	Short: "generate entity",
	Long:  `generate entity`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[gen] ")
		dir, _ := os.Getwd()

		aimPath := filepath.Join(dir, "internal", "data")
		stat, err := os.Stat(aimPath)
		if err != nil || !stat.IsDir() {
			log.Println("please run in root project")
			log.Fatalf("can't find data dir [%s]\n", aimPath)
		}

		env := os.Environ()
		runArgs := []string{"generate", "./ent"}
		log.Println("run args: ", strings.Join(runArgs, " "))

		res, err := core.RunWithDir("go", aimPath, env, runArgs...)
		if err != nil {
			log.SetPrefix("[err]")
		}
		log.Println(res)
		utils.CheckErrWithStatus(err)
		log.Println("generate entity success")

		// featureArr, _ := cmd.Flags().GetStringSlice("feature")
		// idType, _ := cmd.Flags().GetString("idtype")
		// log.SetPrefix("[gen] ")
		// koanf, err := config.NewConfig()
		// utils.CheckErrWithStatus(err)
		//
		// prefix, _ := cmd.Flags().GetString("prefix")
		// if prefix == "" {
		// 	prefix = koanf.String("db.tablePrefix")
		// }
		//
		// cwd, _ := os.Getwd()
		// schemaPath := filepath.Join(cwd, "internal/data/ent/schema")
		//
		// var features []gen.Feature
		// for _, feature := range featureArr {
		// 	switch feature {
		// 	case "schema/snapshot":
		// 		features = append(features, gen.FeatureSnapshot)
		// 	case "sql/modifier":
		// 		features = append(features, gen.FeatureModifier)
		// 	case "sql/versioned-migration":
		// 		features = append(features, gen.FeatureVersionedMigration)
		// 	case "privacy":
		// 		features = append(features, gen.FeaturePrivacy)
		// 	case "entql":
		// 		features = append(features, gen.FeatureEntQL)
		// 	case "sql/schemaconfig":
		// 		features = append(features, gen.FeatureSchemaConfig)
		// 	case "sql/lock":
		// 		features = append(features, gen.FeatureLock)
		// 	case "sql/execquery":
		// 		features = append(features, gen.FeatureExecQuery)
		// 	case "sql/upsert":
		// 		features = append(features, gen.FeatureUpsert)
		// 	case "namedges":
		// 	}
		// }
		//
		// log.Println("features:", featureArr)
		//
		// cfg := &gen.Config{
		// 	Hooks: []gen.Hook{
		// 		PrefixSchema(prefix),
		// 	},
		// 	Features: features,
		// }
		// if idType == "int64" {
		// 	log.Printf("id type: %s\n", idType)
		// 	cfg.IDType = &field.TypeInfo{Type: field.TypeInt64}
		// }
		// err = entc.Generate(schemaPath, cfg)
		// utils.CheckErrWithStatus(err)

	},
}

var genCmdCmd = &cobra.Command{
	Use:   "cmd",
	Short: "generate command",
	Long:  `generate command`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		modPath, err := utils.FindModPath(3)
		utils.CheckErrWithStatus(err)

		modName := utils.GetModName()

		log.Println("now in mod: " + modName)
		NewGenSubCmd(name, modPath, modName).Gen()

	},
}

var genTsCmd = &cobra.Command{
	Use:   "tp",
	Short: "generate types",
	Long:  `generate types`,
	Run: func(cmd *cobra.Command, args []string) {
		//force, _ := cmd.Flags().GetBool("force")
		genTs, _ := cmd.Flags().GetBool("ts")
		genGo, _ := cmd.Flags().GetBool("go")
		clientName, _ := cmd.Flags().GetString("name")

		log.SetPrefix("[gen] ")
		dir, _ := os.Getwd()

		if !genTs && !genGo {
			log.Println("please specify type")
			return
		}

		// 1. 获取当前目录下的build文件夹里面的.log文件
		buildDir := filepath.Join(dir, "build")
		stat, err := os.Stat(buildDir)
		utils.CheckErrWithStatus(err)

		if !stat.IsDir() {
			utils.CheckErrWithStatus(errors.New("build is not a dir"))
		}

		files, err := filex.LoadFiles(buildDir, func(s string) bool {
			return strings.HasSuffix(s, ".log")
		})
		utils.CheckErrWithStatus(err)

		for _, file := range files {
			log.Println("file: ", file)
			if genTs {
				rtn, err := typex.Parse2Ts(file)
				if err != nil {
					log.Printf("parse log file [%s] error: %s\n", file, err.Error())
					continue
				}

				tsFilename := strings.ReplaceAll(file, ".log", ".ts")
				log.Println("tsFilename: ", tsFilename)

				err = typex.WriteGen(tsFilename, rtn)
				utils.CheckErrWithStatus(err)
			}
			if genGo {
				var goRtn []string
				goRtn, err = typex.Parse2Go(file, strcase.ToCamel(clientName))
				if err != nil {
					log.Printf("parse log file [%s] error: %s\n", file, err.Error())
					continue
				}

				goFilename := strings.ReplaceAll(file, ".log", ".go")
				log.Println("goFilename: ", goFilename)

				err = typex.WriteGen(goFilename, goRtn)
				utils.CheckErrWithStatus(err)
			}
		}

	},
}

type GenSubCmd struct {
	Name       string
	ModPath    string // root path
	ModName    string // root mod name
	StructName string // eg: HelloYouCmd
	KebabName  string // eg: hello-you
	SnakeName  string // eg: hello_you

	StructAppName string // eg: HelloYouApp
}

func NewGenSubCmd(name string, modPath string, modName string) *GenSubCmd {
	return &GenSubCmd{Name: name, ModPath: modPath, ModName: modName}
}

func (g *GenSubCmd) Gen() {
	log.SetPrefix("[gen] ")
	log.Println("generate command: ", g.Name)
	g.checkFile()
	g.updateFields()
	g.genCmd()
	g.genCmdApp()
	g.updateWireFile()
}

func (g *GenSubCmd) updateFields() {
	g.StructName = strcase.ToCamel(g.Name) + "Cmd"
	g.StructAppName = strcase.ToCamel(g.Name) + "App"
	g.KebabName = strcase.ToKebab(g.Name)
	g.SnakeName = strcase.ToSnake(g.Name)
}

func (g *GenSubCmd) checkFile() {
	cmdPath := filepath.Join(g.ModPath, "internal/cmd", g.SnakeName+".go")
	if _, err := os.Stat(cmdPath); err == nil {
		utils.CheckErrWithStatus(errors.New("cmd file already exists"))
		return
	}
	cmdAppPath := filepath.Join(g.ModPath, "internal/cmd_app", g.SnakeName+".go")
	if _, err := os.Stat(cmdAppPath); err == nil {
		utils.CheckErrWithStatus(errors.New("cmd_app file already exists"))
		return
	}
}

func (g *GenSubCmd) genCmd() {
	savePath := filepath.Join(g.ModPath, "internal/cmd", g.SnakeName+".go")

	parse, err := template.New("cmd").Parse(tpl.CmdTpl)
	utils.CheckErrWithStatus(err)

	buffer := bytes.NewBuffer([]byte{})
	err = parse.Execute(buffer, g)
	utils.CheckErrWithStatus(err)

	source, err := format.Source(buffer.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(savePath, source, false)
	utils.CheckErrWithStatus(err)
}
func (g *GenSubCmd) genCmdApp() {
	savePath := filepath.Join(g.ModPath, "internal/cmd_app", g.SnakeName+"_app.go")

	parse, err := template.New("cmd_app").Parse(tpl.CmdAppTpl)
	utils.CheckErrWithStatus(err)

	buffer := bytes.NewBuffer([]byte{})
	err = parse.Execute(buffer, g)
	utils.CheckErrWithStatus(err)

	source, err := format.Source(buffer.Bytes())
	utils.CheckErrWithStatus(err)

	err = utils.SaveToFile(savePath, source, false)
	utils.CheckErrWithStatus(err)
}
func (g *GenSubCmd) updateWireFile() {
	log.Println("updating wire file")
	wireFilePath := filepath.Join(g.ModPath, "internal/cmd_app", "wire.go")
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, wireFilePath, nil, parser.ParseComments)
	utils.CheckErrWithStatus(err)

	wk := visitor.NewCmdWireVisitor(g.StructName, g.StructAppName)
	ast.Walk(wk, f)
	wk.InsertInitCmdFunc(f)

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	_ = utils.SaveToFile(wireFilePath, dst.Bytes(), true)

	log.Println("update wire file success")
}

func init() {

	strcase.ConfigureAcronym("neo4j", "neo4j")

	genCmd.AddCommand(genEntCmd, genCmdCmd, genTsCmd)
	rootCmd.AddCommand(genCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// genCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// genCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	genCmd.Flags().StringP("type", "t", "route", "type of gen")
	genCmd.Flags().Bool("no-repo", false, "generate repo file")
	genCmd.Flags().Bool("v2", false, "is v2")
	genCmd.Flags().Bool("with-crud", false, "generate crud section in repo and biz file")
	genCmd.Flags().String("ent-name", "", "generate crud section's ent name")
	genCmd.Flags().StringP("name", "n", "", "name of gen")

	genCmd.MarkFlagsRequiredTogether("type", "name")

	genEntCmd.Flags().StringP("prefix", "p", "", "prefix of entity")
	genEntCmd.Flags().StringP("idtype", "i", "int64", "id type of entity")
	genEntCmd.Flags().StringSliceP("feature", "f", []string{"sql/modifier", "sql/versioned-migration"}, "the features of the ent for generating entity")

	genCmdCmd.Flags().StringP("name", "n", "", "name of gen")
	_ = genCmdCmd.MarkFlagRequired("name")

	//genTsCmd.Flags().BoolP("force", "f", false, "force generate ts interface")
	genTsCmd.Flags().Bool("ts", false, "generate typescript interface")
	genTsCmd.Flags().Bool("go", false, "generate go api client")
	genTsCmd.Flags().StringP("name", "n", "ApiClient", "name of client")

}
