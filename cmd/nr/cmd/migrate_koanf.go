package cmd

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/utils"
)

const (
	legacyKoanfModule = "github.com/knadh/koanf"
	koanfV2Module     = "github.com/knadh/koanf/v2"
	defaultKoanfV2    = "v2.2.2"
)

var migrateKoanfCmd = &cobra.Command{
	Use:   "koanf",
	Short: "check or migrate the legacy github.com/knadh/koanf module to /v2",
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		root, err := projectRoot(dir)
		if err != nil {
			return err
		}
		apply, _ := cmd.Flags().GetBool("apply")
		check, _ := cmd.Flags().GetBool("check")
		skipTidy, _ := cmd.Flags().GetBool("skip-tidy")
		version, _ := cmd.Flags().GetString("version")
		return migrateKoanf(root, apply, check, skipTidy, version)
	},
}

func init() {
	migrateKoanfCmd.Flags().StringP("dir", "d", "", "project directory (default: current directory)")
	migrateKoanfCmd.Flags().Bool("apply", false, "write the Koanf v2 migration; default is preview only")
	migrateKoanfCmd.Flags().Bool("check", false, "fail when a legacy Koanf dependency or import is found")
	migrateKoanfCmd.Flags().Bool("skip-tidy", false, "do not run go mod tidy after applying")
	migrateKoanfCmd.Flags().String("version", defaultKoanfV2, "Koanf v2 module version to require")
	migrateCmd.AddCommand(migrateKoanfCmd)
}

type koanfMigrationPlan struct {
	goFiles []string
	legacy  bool
}

func projectRoot(dir string) (string, error) {
	start, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve project directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(start, "go.mod")); err == nil {
			return start, nil
		}
		parent := filepath.Dir(start)
		if parent == start {
			return "", fmt.Errorf("find project root: %w", utils.ErrNotFoundMod)
		}
		start = parent
	}
}

func migrateKoanf(root string, apply, check, skipTidy bool, version string) error {
	plan, err := planKoanfMigration(root)
	if err != nil {
		return err
	}
	if !plan.legacy {
		fmt.Println("Koanf v2 is already in use; no migration is needed.")
		return nil
	}

	fmt.Printf("Legacy Koanf detected: %d Go import(s) use %s.\n", len(plan.goFiles), legacyKoanfModule)
	if check {
		return fmt.Errorf("legacy Koanf detected; run `nr migrate koanf --apply`")
	}
	if !apply {
		fmt.Println("Preview only. Re-run with --apply to rewrite imports and go.mod.")
		return nil
	}
	if !strings.HasPrefix(version, "v2.") {
		return fmt.Errorf("Koanf v2 version must start with v2., got %q", version)
	}

	for _, path := range plan.goFiles {
		if err := rewriteKoanfImport(path); err != nil {
			return err
		}
	}
	if err := rewriteKoanfGoMod(filepath.Join(root, "go.mod"), version); err != nil {
		return err
	}
	if !skipTidy {
		if output, err := core.RunWithDir("go", root, nil, "mod", "tidy"); err != nil {
			return fmt.Errorf("go mod tidy after Koanf migration: %w\n%s", err, output)
		}
	}
	fmt.Printf("Migrated %d Go import(s) to %s.\n", len(plan.goFiles), koanfV2Module)
	return nil
}

func planKoanfMigration(root string) (koanfMigrationPlan, error) {
	plan := koanfMigrationPlan{}
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return plan, fmt.Errorf("read go.mod: %w", err)
	}
	mod, err := modfile.Parse("go.mod", goMod, nil)
	if err != nil {
		return plan, fmt.Errorf("parse go.mod: %w", err)
	}
	for _, req := range mod.Require {
		if req.Mod.Path == legacyKoanfModule {
			plan.legacy = true
		}
	}

	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_gen.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse imports in %s: %w", path, err)
		}
		for _, imp := range file.Imports {
			if strings.Trim(imp.Path.Value, "\"") == legacyKoanfModule {
				plan.goFiles = append(plan.goFiles, path)
				plan.legacy = true
				break
			}
		}
		return nil
	})
	return plan, err
}

func rewriteKoanfImport(path string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if !astutil.RewriteImport(fset, file, legacyKoanfModule, koanfV2Module) {
		return nil
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	if err := utils.WriteFileAtomic(path, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func rewriteKoanfGoMod(path, version string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	mod, err := modfile.Parse(path, content, nil)
	if err != nil {
		return fmt.Errorf("parse go.mod: %w", err)
	}
	if err := mod.DropRequire(legacyKoanfModule); err != nil {
		return fmt.Errorf("remove legacy Koanf requirement: %w", err)
	}
	if err := mod.AddRequire(koanfV2Module, version); err != nil {
		return fmt.Errorf("add Koanf v2 requirement: %w", err)
	}
	formatted, err := mod.Format()
	if err != nil {
		return fmt.Errorf("format go.mod: %w", err)
	}
	if err := utils.WriteFileAtomic(path, formatted, 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}
	return nil
}
