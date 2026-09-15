/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/internal/wire2loom"
	"github.com/Xwudao/neter/pkg/utils"
)

// loomCmd regenerates every dependency injection graph in the project.
//
// Unlike the Wire command it replaces, this does not hunt for injector files:
// the loom CLI discovers graphs itself, generates all of them before writing
// anything, and can be asked to verify freshness with -dry-run.
var loomCmd = &cobra.Command{
	Use:   "loom",
	Short: "regenerate dependency injection code for every loom graph in the project",
	Long: `Regenerate the loom_gen.go files for every loom.Graph declared in the project.

The loom CLI version is pinned by the "tool" directive in go.mod, so the same
version is used by everyone working on the project and by CI.

Run "nr loom convert" first if the project still uses Wire.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return runLoomGenerate(dryRun)
	},
}

// wireCmd is kept so existing scripts and habits keep working.
var wireCmd = &cobra.Command{
	Use:    "wire",
	Short:  "deprecated alias for `nr loom`",
	Hidden: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		fmt.Println("warning: `nr wire` is deprecated; use `nr loom`.")
		return runLoomGenerate(false)
	},
}

func runLoomGenerate(dryRun bool) error {
	root, err := utils.FindProjectRoot(8)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	args := []string{"tool", "loom", "generate"}
	if dryRun {
		args = append(args, "-dry-run")
	}
	args = append(args, "./...")

	out, err := core.RunWithDir("go", root, nil, args...)
	if out != "" {
		fmt.Print(out)
		if out[len(out)-1] != '\n' {
			fmt.Println()
		}
	}
	if err != nil {
		if dryRun {
			return fmt.Errorf("loom generation is out of date; run `nr loom`: %w", err)
		}
		return fmt.Errorf("loom generate failed: %w\nhint: go.mod must declare `tool github.com/Xwudao/loom/cmd/loom`; run `nr loom convert` to migrate a Wire project", err)
	}
	return nil
}

// loomConvertCmd rewrites a Wire project to Loom.
var loomConvertCmd = &cobra.Command{
	Use:   "convert",
	Short: "rewrite a project's dependency injection from Wire to Loom",
	Long: `Rewrite the Wire declarations of the current project as Loom graphs.

The command converts provider sets (wire.NewSet -> loom.Module), interface
bindings (wire.Bind -> loom.As), injector stubs (wire.Build -> loom.Graph) and
the "value, cleanup, error" call pattern (-> *loom.Lifecycle). It then regenerates
the graphs and builds the project; if anything fails, every file is restored and
nothing is left half-converted.

Anything it cannot convert is reported and aborts the run, so the project is
never left in a state that needs guessing. Use --dry-run to see the plan first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		skipVerify, _ := cmd.Flags().GetBool("skip-verify")
		loomVersion, _ := cmd.Flags().GetString("loom-version")
		dir, _ := cmd.Flags().GetString("dir")

		root := dir
		if root == "" {
			var err error
			root, err = utils.FindProjectRoot(8)
			if err != nil {
				return fmt.Errorf("find project root: %w", err)
			}
		}

		result, err := wire2loom.Convert(wire2loom.Options{
			Root:        root,
			Apply:       !dryRun,
			LoomVersion: loomVersion,
			SkipVerify:  dryRun || skipVerify,
		})
		if err != nil {
			return err
		}
		printConversion(root, result)
		if dryRun {
			fmt.Println("\ndry run: nothing was written; re-run without --dry-run to apply")
		}
		return nil
	},
}

func printConversion(root string, result *wire2loom.Result) {
	for _, path := range result.Rewritten {
		fmt.Printf("rewritten  %s\n", path)
	}
	for _, path := range result.Created {
		fmt.Printf("created    %s\n", path)
	}
	for _, path := range result.Removed {
		fmt.Printf("removed    %s\n", path)
	}
	dirs := make([]string, 0, len(result.Graphs))
	for dir := range result.Graphs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			rel = dir
		}
		fmt.Printf("graph      %s: %s\n", filepath.ToSlash(rel), strings.Join(result.Graphs[dir], ", "))
	}
	for _, note := range result.Notes {
		fmt.Printf("note       %s\n", note)
	}
}

func init() {
	rootCmd.AddCommand(loomCmd)
	rootCmd.AddCommand(wireCmd)

	loomCmd.AddCommand(loomConvertCmd)

	loomConvertCmd.Flags().Bool("dry-run", false, "print the conversion plan without writing anything")
	loomConvertCmd.Flags().Bool("skip-verify", false, "skip go mod tidy, graph generation and the build (no safety net)")
	loomConvertCmd.Flags().String("loom-version", "latest", "github.com/Xwudao/loom version to depend on")
	loomConvertCmd.Flags().StringP("dir", "d", "", "project root (default: nearest go.mod)")

	loomCmd.Flags().Bool("dry-run", false, "fail instead of writing when a generated file is out of date")
	_ = wireCmd.Flags().Bool("dry-run", false, "fail instead of writing when a generated file is out of date")
}
