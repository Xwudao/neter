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

	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/entc/gen"
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
	Short: "gen some for nr",
	Long:  `gen route, service, or other things for nr`,
	Run: func(cmd *cobra.Command, args []string) {
		tpe, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")
		noRepo, _ := cmd.Flags().GetBool("no-repo")
		withCrud, _ := cmd.Flags().GetBool("with-crud")
		entName, _ := cmd.Flags().GetString("ent-name")

		g := NewGenerate(name, tpe, withCrud, entName)

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

	StructRouteName string // eg: HomeRoute
	StructBizName   string // eg: HomeBiz
	StructRepoName  string // eg: HomeRepo
}

func NewGenerate(name string, typeName string, crud bool, entName string) *GenerateRoute {
	return &GenerateRoute{Name: name, TypeName: typeName, WithCRUD: crud, EntName: entName}
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
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	// update content
	walker := visitor.NewUpdateRoot(fmt.Sprintf("%s%s", g.PackageName, g.StructRouteName), fmt.Sprintf("*%s.%s", g.PackageName, g.StructRouteName))
	ast.Walk(walker, f)

	// update imports
	pkgName := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	_ = astutil.AddNamedImport(fset, f, g.PackageName, pkgName)
	// if !added {
	//	utils.CheckErrWithStatus(fmt.Errorf("can't add import [%s]", pkgName))
	// }

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	err = utils.SaveToFile(rootFilePath, dst.Bytes(), true)
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
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	// update content
	//walker := visitor.NewCurrentProvideVisitor(fmt.Sprintf("New%s", g.StructBizName))
	//ast.Walk(walker, f)
	visitor.UpdateProvider(f, "ProviderBizSet", fmt.Sprintf("New%s", g.StructBizName))

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	err = utils.SaveToFile(rootFilePath, dst.Bytes(), true)
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
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	// update content
	//walker := visitor.NewCurrentProvideVisitor(fmt.Sprintf("New%s", g.StructRepoName))
	//ast.Walk(walker, f)
	visitor.UpdateProvider(f, "ProviderDataSet", fmt.Sprintf("New%s", g.StructRepoName))

	var dst bytes.Buffer
	err = format.Node(&dst, fset, f)
	utils.CheckErrWithStatus(err)
	err = utils.SaveToFile(rootFilePath, dst.Bytes(), true)
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
	f, err := parser.ParseFile(fset, rootFilePath, nil, 0)
	utils.CheckErrWithStatus(err)

	// update content
	walker := visitor.NewRouteProvideVisitor(g.PackageName, fmt.Sprintf("New%s", g.StructRouteName))
	ast.Walk(walker, f)

	// update imports
	pkgName := fmt.Sprintf("%s/internal/routes/%s", g.ModName, g.PackageName)
	_ = astutil.AddNamedImport(fset, f, g.PackageName, pkgName)
	// if !added {
	//	utils.CheckErrWithStatus(fmt.Errorf("can't add import [%s]", pkgName))
	// }

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
		runArgs := []string{"generate", "./ent", "-run", "entgo.io/ent/cmd/ent"}
		log.Println("run args: ", strings.Join(runArgs, " "))

		res, err := runWithDir("go", aimPath, env, runArgs...)
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

var genCmdCmd = &cobra.Command{
	Use:   "cmd",
	Short: "generate command",
	Long:  `generate command`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		modPath, err := utils.FindModPath(3)
		utils.CheckErrWithStatus(err)

		NewGenSubCmd(name, modPath).Gen()

	},
}

type GenSubCmd struct {
	Name       string
	ModPath    string // root path
	StructName string // eg: HelloYouCmd
	KebabName  string // eg: hello-you
}

func NewGenSubCmd(name string, modPath string) *GenSubCmd {
	return &GenSubCmd{Name: name, ModPath: modPath}
}

func (g *GenSubCmd) Gen() {
	log.SetPrefix("[gen] ")
	log.Println("generate command: ", g.Name)
	g.checkFile()
	g.genCmd()
}

func (g *GenSubCmd) checkFile() {
	snakeName := strcase.ToSnake(g.Name)
	cmdPath := filepath.Join(g.ModPath, "internal/cmd", snakeName+".go")
	if _, err := os.Stat(cmdPath); err == nil {
		utils.CheckErrWithStatus(errors.New("file already exists"))
		return
	}
}

func (g *GenSubCmd) genCmd() {
	g.StructName = strcase.ToCamel(g.Name) + "Cmd"
	g.KebabName = strcase.ToKebab(g.Name)
	savePath := filepath.Join(g.ModPath, "internal/cmd", g.KebabName+".go")

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

func init() {

	strcase.ConfigureAcronym("neo4j", "neo4j")

	genCmd.AddCommand(genEntCmd, genCmdCmd)
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
	genCmd.Flags().Bool("with-crud", false, "generate crud section in repo and biz file")
	genCmd.Flags().String("ent-name", "", "generate crud section's ent name")
	genCmd.Flags().StringP("name", "n", "", "name of gen")

	genCmd.MarkFlagsRequiredTogether("type", "name")

	genEntCmd.Flags().StringP("prefix", "p", "", "prefix of entity")
	genEntCmd.Flags().StringP("idtype", "i", "int64", "id type of entity")
	genEntCmd.Flags().StringSliceP("feature", "f", []string{"sql/modifier", "sql/versioned-migration"}, "the features of the ent for generating entity")

	genCmdCmd.Flags().StringP("name", "n", "", "name of gen")
	_ = genCmdCmd.MarkFlagRequired("name")

}
