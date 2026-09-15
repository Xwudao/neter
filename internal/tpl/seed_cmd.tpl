package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"{{.ModName}}/internal/cmd_app"
	"{{.ModName}}/internal/seed"
)

var seedCmd = &cobra.Command{
	Use:   "seed [name ...]",
	Short: "initialize idempotent application data",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, names []string) error {
		app, cleanup, err := cmd_app.SeedCmd()
		if err != nil {
			return err
		}
		defer cleanup()

		list, err := cmd.Flags().GetBool("list")
		if err != nil {
			return err
		}
		if list {
			for _, name := range app.Names() {
				fmt.Fprintln(cmd.OutOrStdout(), name)
			}
			return nil
		}

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		run, err := app.Run(names, seed.Options{DryRun: dryRun, Force: force})
		if err != nil {
			return err
		}
		for _, name := range run {
			fmt.Fprintf(cmd.OutOrStdout(), "seeded %s\n", name)
		}
		return nil
	},
}

func init() {
	seedCmd.Flags().Bool("list", false, "list available seeders without running them")
	seedCmd.Flags().Bool("dry-run", false, "validate seeders without writing data when supported")
	seedCmd.Flags().Bool("force", false, "allow seeders to replace operator-managed data when supported")
	rootCmd.AddCommand(seedCmd)
}
