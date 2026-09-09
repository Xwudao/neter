package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/internal/migrate"
)

var migrateNewCmd = &cobra.Command{
	Use:     "new <name>",
	Aliases: []string{"create"},
	Short:   "create a paired <version>_<name>.up.sql / .down.sql migration",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := migrateDBOptions(cmd)
		if err != nil {
			return err
		}
		created, err := migrate.Create(o.Dir, args[0])
		if err != nil {
			return err
		}
		fmt.Printf("created %s\n", created.UpPath)
		fmt.Printf("created %s\n", created.DownPath)
		fmt.Println("edit both files, then run `nr migrate up` to apply them")
		return nil
	},
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "apply pending database migrations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := migrateDBOptions(cmd)
		if err != nil {
			return err
		}
		warnLegacyEntProject()
		steps, _ := cmd.Flags().GetInt("steps")
		status, err := o.Up(steps)
		if err != nil {
			return err
		}
		printMigrateStatus("up", o.Dir, status)
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "roll back database migrations",
	Long: `Roll back the most recent migration. Use --all to roll back every applied
migration and --steps N to roll back N migrations.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := migrateDBOptions(cmd)
		if err != nil {
			return err
		}
		warnLegacyEntProject()
		all, _ := cmd.Flags().GetBool("all")
		steps, _ := cmd.Flags().GetInt("steps")
		status, err := o.Down(steps, all)
		if err != nil {
			return err
		}
		printMigrateStatus("down", o.Dir, status)
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "show the current database migration version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := migrateDBOptions(cmd)
		if err != nil {
			return err
		}
		status, err := o.Status()
		if err != nil {
			return err
		}
		if !status.HasVersion {
			fmt.Printf("no migration applied yet (%s)\n", o.Dir)
			return nil
		}
		dirty := "clean"
		if status.Dirty {
			dirty = "DIRTY"
		}
		fmt.Printf("version: %d (%s)\n", status.Version, dirty)
		if status.Dirty {
			return fmt.Errorf("database is dirty at version %d; fix it with `nr migrate force <version>`", status.Version)
		}
		return nil
	},
}

var migrateForceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "set the schema version without running migrations (dirty recovery)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o, err := migrateDBOptions(cmd)
		if err != nil {
			return err
		}
		var version int
		if _, err := fmt.Sscanf(args[0], "%d", &version); err != nil {
			return fmt.Errorf("invalid version %q: %w", args[0], err)
		}
		status, err := o.Force(version)
		if err != nil {
			return err
		}
		printMigrateStatus("force", o.Dir, status)
		return nil
	},
}

func bindMigrateDBFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("dir", "d", "", "project root (default: current directory)")
	cmd.Flags().String("path", "", "migration directory relative to the project root (default: db/migrations)")
	cmd.Flags().String("dsn", "", "database DSN (overrides NETER_DSN/DATABASE_URL and config.yml)")
	cmd.Flags().String("config", "", "config.yml path used for the DSN fallback (default: <root>/config.yml)")
	cmd.Flags().String("dialect", "", "database dialect: postgres|mysql (default: from config.yml)")
}

func migrateDBOptions(cmd *cobra.Command) (*migrate.Options, error) {
	root, _ := cmd.Flags().GetString("dir")
	dir, _ := cmd.Flags().GetString("path")
	dsn, _ := cmd.Flags().GetString("dsn")
	configFile, _ := cmd.Flags().GetString("config")
	dialect, _ := cmd.Flags().GetString("dialect")
	return migrate.NewOptions(root, dir, dsn, configFile, dialect)
}

// warnLegacyEntProject reminds Ent users that file-based migrations only apply
// to the new sqlc stack. It is intentionally non-fatal so a mixed project can
// still run nr migrate once db/migrations exists.
func warnLegacyEntProject() {
	if core.DetectCurrentProjectKind().IsEnt() {
		fmt.Println("note: legacy Ent project detected; `nr migrate` only runs file-based migrations under db/migrations")
	}
}

func printMigrateStatus(action, dir string, status migrate.Status) {
	switch {
	case !status.HasVersion:
		fmt.Printf("%s ok: no migration version recorded (%s)\n", action, dir)
	case status.Dirty:
		fmt.Printf("%s ok: version %d (DIRTY) (%s)\n", action, status.Version, dir)
	default:
		fmt.Printf("%s ok: version %d (%s)\n", action, status.Version, dir)
	}
}

func init() {
	for _, cmd := range []*cobra.Command{migrateNewCmd, migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateForceCmd} {
		bindMigrateDBFlags(cmd)
	}
	migrateUpCmd.Flags().Int("steps", 0, "apply at most N pending migrations (0 = all)")
	migrateDownCmd.Flags().Bool("all", false, "roll back every applied migration")
	migrateDownCmd.Flags().Int("steps", 1, "number of migrations to roll back")

	migrateCmd.AddCommand(migrateNewCmd, migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateForceCmd)
}
