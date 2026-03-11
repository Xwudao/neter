package cmd

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/Xwudao/neter/internal/core"
	internalgen "github.com/Xwudao/neter/internal/gen"
	"github.com/Xwudao/neter/pkg/filex"
	"github.com/Xwudao/neter/pkg/typex"
	"github.com/Xwudao/neter/pkg/utils"
)

const (
	genTypeRoute = "route"
	genTypeBiz   = "biz"
)

var legacyGenFlagNames = []string{"type", "name", "no-repo", "v2", "with-crud", "with-params", "ent-name"}

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "generate code and assets",
	Long:  `Generate routes, biz scaffolds, entities, commands, and types for nr.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.CheckErrWithStatus(runLegacyGen(cmd))
	},
}

var genRouteCmd = &cobra.Command{
	Use:   "route",
	Short: "generate a route scaffold",
	Run: func(cmd *cobra.Command, args []string) {
		utils.CheckErrWithStatus(runTypedGenerator(cmd, genTypeRoute))
	},
}

var genBizCmd = &cobra.Command{
	Use:   "biz",
	Short: "generate a biz scaffold",
	Run: func(cmd *cobra.Command, args []string) {
		utils.CheckErrWithStatus(runTypedGenerator(cmd, genTypeBiz))
	},
}

func runLegacyGen(cmd *cobra.Command) error {
	usedLegacyFlags := false
	for _, flagName := range legacyGenFlagNames {
		if cmd.Flags().Changed(flagName) {
			usedLegacyFlags = true
			break
		}
	}

	if !usedLegacyFlags {
		return cmd.Help()
	}

	typeName, _ := cmd.Flags().GetString("type")
	cmd.PrintErrln("warning: `nr gen --type ...` is deprecated; use `nr gen route` or `nr gen biz`.")
	return runTypedGenerator(cmd, typeName)
}

func runTypedGenerator(cmd *cobra.Command, typeName string) error {
	req, err := newGenRequest(cmd, typeName)
	if err != nil {
		return err
	}

	return internalgen.Execute(req)
}

func newGenRequest(cmd *cobra.Command, typeName string) (internalgen.Request, error) {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return internalgen.Request{}, errors.New("please specify --name")
	}

	req := internalgen.Request{
		TypeName: typeName,
		Name:     name,
	}

	switch typeName {
	case genTypeRoute:
		req.V2, _ = cmd.Flags().GetBool("v2")
	case genTypeBiz:
		req.NoRepo, _ = cmd.Flags().GetBool("no-repo")
		req.WithCRUD, _ = cmd.Flags().GetBool("with-crud")
		req.WithParams, _ = cmd.Flags().GetBool("with-params")
		req.EntName, _ = cmd.Flags().GetString("ent-name")
	case "":
		return internalgen.Request{}, errors.New("please specify a generator type")
	default:
		return internalgen.Request{}, errors.New("unknown type")
	}

	return req, nil
}

func bindNameFlag(flags *pflag.FlagSet) {
	flags.StringP("name", "n", "", "name of gen")
}

func bindRouteFlags(flags *pflag.FlagSet) {
	bindNameFlag(flags)
	flags.Bool("v2", false, "use v2 route template")
}

func bindBizFlags(flags *pflag.FlagSet) {
	bindNameFlag(flags)
	flags.Bool("no-repo", false, "skip repo file generation")
	flags.Bool("with-crud", false, "generate crud section in repo and biz file")
	flags.Bool("with-params", false, "generate params file")
	flags.String("ent-name", "", "generate crud section's ent name")
}

func hideLegacyGenFlags(cmd *cobra.Command) {
	for _, flagName := range legacyGenFlagNames {
		_ = cmd.Flags().MarkHidden(flagName)
	}
}

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
	},
}

var genCmdCmd = &cobra.Command{
	Use:   "cmd",
	Short: "generate command scaffolding",
	Long:  `generate command scaffolding`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		modPath, err := utils.FindModPath(3)
		utils.CheckErrWithStatus(err)

		modName := utils.GetModName()

		log.Println("now in mod: " + modName)
		utils.CheckErrWithStatus(internalgen.NewSubCmd(name, modPath, modName).Gen())
	},
}

var genTsCmd = &cobra.Command{
	Use:     "tp",
	Aliases: []string{"types"},
	Short:   "generate types",
	Long:    `generate types`,
	Run: func(cmd *cobra.Command, args []string) {
		genTs, _ := cmd.Flags().GetBool("ts")
		genGo, _ := cmd.Flags().GetBool("go")
		clientName, _ := cmd.Flags().GetString("name")

		log.SetPrefix("[gen] ")
		dir, _ := os.Getwd()

		if !genTs && !genGo {
			log.Println("please specify type")
			return
		}

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
				goRtn, err := typex.Parse2Go(file, strcase.ToCamel(clientName))
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

func init() {
	strcase.ConfigureAcronym("neo4j", "neo4j")

	genCmd.AddCommand(genRouteCmd, genBizCmd, genEntCmd, genCmdCmd, genTsCmd)
	rootCmd.AddCommand(genCmd)

	genCmd.Flags().StringP("type", "t", genTypeRoute, "type of gen")
	bindBizFlags(genCmd.Flags())
	genCmd.Flags().Bool("v2", false, "use v2 route template")
	hideLegacyGenFlags(genCmd)

	bindRouteFlags(genRouteCmd.Flags())
	_ = genRouteCmd.MarkFlagRequired("name")

	bindBizFlags(genBizCmd.Flags())
	_ = genBizCmd.MarkFlagRequired("name")

	genEntCmd.Flags().StringP("prefix", "p", "", "prefix of entity")
	genEntCmd.Flags().StringP("idtype", "i", "int64", "id type of entity")
	genEntCmd.Flags().StringSliceP("feature", "f", []string{"sql/modifier", "sql/versioned-migration"}, "the features of the ent for generating entity")

	genCmdCmd.Flags().StringP("name", "n", "", "name of gen")
	_ = genCmdCmd.MarkFlagRequired("name")

	genTsCmd.Flags().Bool("ts", false, "generate typescript interface")
	genTsCmd.Flags().Bool("go", false, "generate go api client")
	genTsCmd.Flags().StringP("name", "n", "ApiClient", "name of client")
}
