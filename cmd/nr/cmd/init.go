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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/pkg/utils"
)

const (
	gitUrl  = "git@github.com:Xwudao/nr-template.git"
	httpUrl = "https://github.com/Xwudao/neter-template.git"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init new project from template",
	Long:  `You can use this command fork a new project from our template.`,
	Run: func(cmd *cobra.Command, args []string) {
		newModName, _ := cmd.Flags().GetString("newMod")
		git, _ := cmd.Flags().GetBool("git")
		i := NewInitProject(newModName, git)
		i.init(args)
		i.clone()
		i.rewriteMod()
		i.rmGit()
		i.modTidy()

		utils.Info("finished, happy hacking!")

	},
}

type InitProject struct {
	projectName string //eg: neter-template
	rootPath    string //eg: /Users/xwudao/go/src/github.com/Xwudao/neter-template
	modPath     string //eg: github.com/Xwudao/neter-template/go.mod

	originModName string //eg: github.com/Xwudao/neter-template
	newModName    string //eg: github.com/Xwudao/new-project
	useGit        bool   //use git schema to clone project or http schema
}

func NewInitProject(newModName string, git bool) *InitProject {
	return &InitProject{newModName: newModName, useGit: git}
}

// init
func (i *InitProject) init(args []string) {
	if len(args) == 0 {
		utils.CheckErrWithStatus(errors.New("please input the project name"))
	}
	i.projectName = args[0]

	dir, _ := os.Getwd()
	i.rootPath = filepath.Join(dir, i.projectName)
	i.modPath = filepath.Join(i.rootPath, "go.mod")

	_, err := os.Stat(i.rootPath)
	if err == nil {
		utils.CheckErrWithStatus(fmt.Errorf("maybe %s path existed, please rename or remove it", i.rootPath))
	}
}

// clone
func (i *InitProject) clone() {
	schemaUrl := httpUrl
	if i.useGit {
		schemaUrl = gitUrl
	}
	utils.Info("cloning project....")
	utils.Info(i.projectName)
	utils.Info(schemaUrl)
	cmd := exec.Command("git", "clone", schemaUrl, i.projectName)
	err := cmd.Run()
	utils.CheckErrWithStatus(err)
	utils.Info("cloned project....")
}

func (i *InitProject) rewriteMod() {
	if i.newModName == "" {
		i.newModName = i.projectName
	}
	var err error
	i.originModName, err = i.getOriginName()
	utils.CheckErrWithStatus(err)
	files := utils.LoadFiles(i.rootPath, func(filename string) bool {
		return path.Ext(filename) == ".go" && !strings.Contains(filename, "/vendor/")
	})
	utils.Info("changing mod name...")
	for _, f := range files {
		node, fset, err := i.parse(f)
		if err != nil {
			utils.Error(err)
			continue
		}
		err = i.write(f, node, fset)
		if err != nil {
			utils.Error(err)
			continue
		}
	}
	err = i.setModName()
	utils.CheckErrWithStatus(err)
	utils.Info("changed mod name")
}
func (i *InitProject) write(filename string, node *ast.File, fset *token.FileSet) error {

	var buf bytes.Buffer

	err := format.Node(&buf, fset, node)
	if err != nil {
		return err
	}

	if filename == "" {
		return fmt.Errorf("no file name")
	}

	err = ioutil.WriteFile(filename, buf.Bytes(), os.ModePerm)
	if err != nil {
		return fmt.Errorf("write file err: %s", err.Error())
	}

	return nil
}
func (i *InitProject) rmGit() {
	gitDir := filepath.Join(i.rootPath, ".git")
	_ = os.RemoveAll(gitDir)
}
func (i *InitProject) parse(filename string) (*ast.File, *token.FileSet, error) {

	fileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(fileSet, filename, nil, parser.ParseComments)

	if err != nil {
		return nil, nil, err
	}

	fset := fileSet
	//astutil.RewriteImport(fset, astFile, p.originModName, p.newModName)

	for _, importSpec := range astFile.Imports {
		originPath := importSpec.Path.Value
		importSpec.Path.Value = strings.Replace(originPath, i.originModName, i.newModName, 1)
	}

	return astFile, fset, nil
}

func (i *InitProject) getOriginName() (name string, err error) {
	_, err = os.Stat(i.modPath)
	if err != nil {
		return
	}

	cnt, err := ioutil.ReadFile(i.modPath)
	if err != nil {
		return
	}

	compile := regexp.MustCompile("(?m)module\\s([^\\s]+)")
	matches := compile.FindStringSubmatch(string(cnt))
	if len(matches) >= 2 {
		return matches[1], nil
	}
	return
}
func (i *InitProject) setModName() (err error) {
	_, err = os.Stat(i.modPath)
	if err != nil {
		return
	}

	cnt, err := ioutil.ReadFile(i.modPath)
	if err != nil {
		return
	}
	nCnt := strings.Replace(string(cnt), i.originModName, i.newModName, 1)
	err = ioutil.WriteFile(i.modPath, []byte(nCnt), os.ModePerm)
	if err != nil {
		return
	}
	return nil
}

func (i *InitProject) modTidy() {
	cmd := exec.Command("go", "mod", "tidy")
	_ = cmd.Run()
}
func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	initCmd.Flags().StringP("newMod", "m", "", "the module name/path")
	initCmd.Flags().BoolP("git", "g", false, "use git method or http method")
}
