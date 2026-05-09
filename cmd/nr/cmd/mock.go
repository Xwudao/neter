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
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/tpl"
	"github.com/Xwudao/neter/pkg/utils"
)

// MockTemplateData is the data passed to biz_test.tpl when generating test stubs.
type MockTemplateData struct {
	ModName       string
	Name          string // raw snake_case base name, e.g. "user"
	StructBizName string // e.g. "UserBiz"
	MockTypeName  string // e.g. "UserRepository" or "RedisRepo"
}

// ────────────────────────────────────────────────────────────────────────────
// Command definition
// ────────────────────────────────────────────────────────────────────────────

var genMockCmd = &cobra.Command{
	Use:   "mock",
	Short: "generate gomock mocks for biz interface files",
	Long: `Scan internal/biz/*_biz.go files for interface definitions and generate
gomock mocks via mockgen. Optionally create test stub files too.

Requires mockgen to be installed:
  go install go.uber.org/mock/mockgen@latest

Examples:
  # Generate mocks for all *_biz.go files
  nr gen mock

  # Generate mock for a single biz file (user_biz.go)
  nr gen mock --name user

  # Generate mocks AND test stub files
  nr gen mock --with-test`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.CheckErrWithStatus(runGenMock(cmd))
	},
}

func init() {
	// Register as a sub-command of genCmd (defined in gen.go).
	genCmd.AddCommand(genMockCmd)
	genMockCmd.Flags().StringP("name", "n", "", "biz name (e.g. user); processes all *_biz.go files if omitted")
	genMockCmd.Flags().Bool("with-test", false, "also generate a test stub file for each mock")
}

// ────────────────────────────────────────────────────────────────────────────
// Core logic
// ────────────────────────────────────────────────────────────────────────────

func runGenMock(cmd *cobra.Command) error {
	log.SetPrefix("[mock] ")

	name, _ := cmd.Flags().GetString("name")
	withTest, _ := cmd.Flags().GetBool("with-test")

	// Verify mockgen is on PATH.
	if _, err := exec.LookPath("mockgen"); err != nil {
		return errors.New("mockgen not found; install it with:\n  go install go.uber.org/mock/mockgen@latest")
	}

	root, err := utils.FindProjectRoot(8)
	if err != nil {
		return fmt.Errorf("cannot find project root (go.mod): %w", err)
	}
	modName := utils.GetModName()

	bizDir := filepath.Join(root, "internal", "biz")
	mocksDir := filepath.Join(bizDir, "mocks")

	if err := os.MkdirAll(mocksDir, 0o755); err != nil {
		return fmt.Errorf("cannot create mocks directory: %w", err)
	}

	// Collect target biz files.
	bizFiles, err := collectBizFiles(bizDir, name)
	if err != nil {
		return err
	}
	if len(bizFiles) == 0 {
		log.Println("no *_biz.go files found")
		return nil
	}

	var genDirectives []string

	for _, bizFile := range bizFiles {
		if !fileHasInterface(bizFile) {
			log.Printf("skip %s (no interface definition found)\n", filepath.Base(bizFile))
			continue
		}

		baseName := strings.TrimSuffix(filepath.Base(bizFile), "_biz.go")
		destFile := filepath.Join(mocksDir, "mock_"+baseName+"_repository.go")

		// mockgen expects POSIX-style paths; convert on Windows.
		relSource := filepath.ToSlash(relPath(root, bizFile))
		relDest := filepath.ToSlash(relPath(root, destFile))

		log.Printf("generating mock: %s → %s\n", relSource, relDest)

		c := exec.Command("mockgen",
			"-source="+relSource,
			"-destination="+relDest,
			"-package=mocks",
		)
		c.Dir = root
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("mockgen failed for %s: %w", relSource, err)
		}

		genDirectives = append(genDirectives, fmt.Sprintf(
			"//go:generate mockgen -source=../%s_biz.go -destination=mock_%s_repository.go -package=mocks",
			baseName, baseName,
		))

		if withTest {
			if err := maybeWriteTestStub(bizDir, baseName, modName); err != nil {
				log.Printf("warning: test stub for %s: %v\n", baseName, err)
			}
		}

		log.Printf("done: %s\n", filepath.Base(destFile))
	}

	if err := writeMockGenGo(mocksDir, genDirectives); err != nil {
		return fmt.Errorf("cannot write mock_gen.go: %w", err)
	}

	log.Println("all mocks generated. To regenerate: go generate ./internal/biz/mocks/...")
	return nil
}

// collectBizFiles returns the list of *_biz.go files to process.
func collectBizFiles(bizDir, name string) ([]string, error) {
	if name != "" {
		p := filepath.Join(bizDir, strcase.ToSnake(name)+"_biz.go")
		if !utils.CheckExist(p) {
			return nil, fmt.Errorf("biz file not found: %s", p)
		}
		return []string{p}, nil
	}

	entries, err := os.ReadDir(bizDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read biz directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_biz.go") {
			files = append(files, filepath.Join(bizDir, e.Name()))
		}
	}
	return files, nil
}

// fileHasInterface reports whether the Go source file contains at least one
// interface type declaration.
func fileHasInterface(path string) bool {
	_, err := findPreferredInterfaceName(path)
	return err == nil
}

func findPreferredInterfaceName(path string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return "", err
	}

	firstInterface := ""
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, isIface := ts.Type.(*ast.InterfaceType); !isIface {
				continue
			}

			name := ts.Name.Name
			if firstInterface == "" {
				firstInterface = name
			}
			if strings.HasSuffix(name, "Repository") || strings.HasSuffix(name, "Repo") {
				return name, nil
			}
		}
	}

	if firstInterface == "" {
		return "", errors.New("no interface definition found")
	}

	return firstInterface, nil
}

// writeMockGenGo writes (or overwrites) mocks/mock_gen.go with package doc
// and //go:generate directives so that `go generate` can regenerate the mocks.
func writeMockGenGo(mocksDir string, directives []string) error {
	var sb strings.Builder
	sb.WriteString("// Package mocks contains generated GoMock mocks for the biz layer.\n")
	sb.WriteString("//\n")
	sb.WriteString("// To regenerate all mocks from the project root:\n")
	sb.WriteString("//\n")
	sb.WriteString("//\tnr gen mock\n")
	sb.WriteString("//\n")
	sb.WriteString("// Or via go generate:\n")
	sb.WriteString("//\n")
	sb.WriteString("//\tgo generate ./internal/biz/mocks/...\n")
	for _, d := range directives {
		sb.WriteString(d + "\n")
	}
	sb.WriteString("package mocks\n")

	return utils.SaveToFile(filepath.Join(mocksDir, "mock_gen.go"), []byte(sb.String()), true)
}

// maybeWriteTestStub generates a test stub file for baseName_biz if it does
// not already exist. Existing test files are never overwritten.
func maybeWriteTestStub(bizDir, baseName, modName string) error {
	testFile := filepath.Join(bizDir, baseName+"_biz_test.go")
	if utils.CheckExist(testFile) {
		log.Printf("skip test stub %s (already exists)\n", filepath.Base(testFile))
		return nil
	}

	bizFile := filepath.Join(bizDir, baseName+"_biz.go")
	mockTypeName, err := findPreferredInterfaceName(bizFile)
	if err != nil {
		return fmt.Errorf("find interface in %s: %w", filepath.Base(bizFile), err)
	}

	data := &MockTemplateData{
		ModName:       modName,
		Name:          baseName,
		StructBizName: strcase.ToCamel(baseName) + "Biz",
		MockTypeName:  mockTypeName,
	}

	parsed, err := template.New("biz_test").Parse(tpl.BizTestTpl)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err := parsed.Execute(buf, data); err != nil {
		return err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format error: %w\n--- source ---\n%s", err, buf.String())
	}

	if err := utils.SaveToFile(testFile, src, false); err != nil {
		return err
	}

	log.Printf("test stub created: %s\n", filepath.Base(testFile))
	return nil
}

// relPath returns target relative to base, or target unchanged on error.
func relPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
