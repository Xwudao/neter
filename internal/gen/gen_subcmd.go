package gen

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/iancoleman/strcase"

	"github.com/Xwudao/neter/internal/tpl"
	"github.com/Xwudao/neter/internal/visitor"
	"github.com/Xwudao/neter/pkg/utils"
)

type GenSubCmd struct {
	Name            string
	ModPath         string
	ModName         string
	StructName      string
	LowerStructName string
	KebabName       string
	SnakeName       string
	StructAppName   string
}

func NewSubCmd(name string, modPath string, modName string) *GenSubCmd {
	return &GenSubCmd{Name: name, ModPath: modPath, ModName: modName}
}

func (g *GenSubCmd) Gen() error {
	log.SetPrefix("[gen] ")
	log.Println("generate command: ", g.Name)
	if err := g.checkFile(); err != nil {
		return err
	}
	g.updateFields()
	if err := g.genCmd(); err != nil {
		return err
	}
	if err := g.genCmdApp(); err != nil {
		return err
	}
	return g.updateWireFile()
}

func (g *GenSubCmd) updateFields() {
	g.StructName = strcase.ToCamel(g.Name) + "Cmd"
	g.LowerStructName = strcase.ToLowerCamel(g.Name) + "Cmd"
	g.StructAppName = strcase.ToCamel(g.Name) + "App"
	g.KebabName = strcase.ToKebab(g.Name)
	g.SnakeName = strcase.ToSnake(g.Name)
}

func (g *GenSubCmd) checkFile() error {
	cmdPath := filepath.Join(g.ModPath, "internal/cmd", g.SnakeName+".go")
	if _, err := os.Stat(cmdPath); err == nil {
		return errors.New("cmd file already exists")
	}

	cmdAppPath := filepath.Join(g.ModPath, "internal/cmd_app", g.SnakeName+".go")
	if _, err := os.Stat(cmdAppPath); err == nil {
		return errors.New("cmd_app file already exists")
	}

	return nil
}

func (g *GenSubCmd) genCmd() error {
	savePath := filepath.Join(g.ModPath, "internal/cmd", g.SnakeName+".go")
	return g.renderTemplateToFile("cmd", tpl.CmdTpl, savePath)
}

func (g *GenSubCmd) genCmdApp() error {
	savePath := filepath.Join(g.ModPath, "internal/cmd_app", g.SnakeName+"_app.go")
	return g.renderTemplateToFile("cmd_app", tpl.CmdAppTpl, savePath)
}

func (g *GenSubCmd) renderTemplateToFile(name string, tplText string, savePath string) error {
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

func (g *GenSubCmd) updateWireFile() error {
	log.Println("updating wire file")
	wireFilePath := filepath.Join(g.ModPath, "internal/cmd_app", "wire.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, wireFilePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	wk := visitor.NewCmdWireVisitor(g.LowerStructName, g.StructName, g.StructAppName)
	ast.Walk(wk, f)
	wk.InsertInitCmdFunc(f)

	var dst bytes.Buffer
	if err := format.Node(&dst, fset, f); err != nil {
		return err
	}
	if err := utils.SaveToFile(wireFilePath, dst.Bytes(), true); err != nil {
		return err
	}

	log.Println("update wire file success")
	return nil
}
