package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/internal/skills"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "generate .agents/skills/<name>/SKILL.md for this project",
	Long: `Generate an agent skill that documents the nr CLI and the commands that
apply to this project's persistence stack.

The file is generated and is overwritten on the next run, so do not edit it.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		name, _ := cmd.Flags().GetString("name")
		kindFlag, _ := cmd.Flags().GetString("kind")
		printOnly, _ := cmd.Flags().GetBool("print")

		root, err := skills.ResolveRoot(dir)
		if err != nil {
			return err
		}

		kind, err := resolveSkillKind(root, kindFlag)
		if err != nil {
			return err
		}

		if printOnly {
			content, err := skills.Render(name, root, kind)
			if err != nil {
				return err
			}
			fmt.Print(content)
			return nil
		}

		path, err := skills.Write(root, name, kind)
		if err != nil {
			return err
		}
		logCommandSuccess("skills", "wrote %s (stack: %s)", path, kind)
		fmt.Println("generated file: do not edit it, `nr skills` will overwrite it")
		return nil
	},
}

// resolveSkillKind turns the --kind flag into a project kind. "auto" (or empty)
// uses the same detection as the rest of the CLI.
func resolveSkillKind(root, flag string) (core.ProjectKind, error) {
	switch flag {
	case "", "auto":
		return core.DetectProjectKind(root), nil
	case string(core.ProjectKindSQLC):
		return core.ProjectKindSQLC, nil
	case string(core.ProjectKindEnt):
		return core.ProjectKindEnt, nil
	case string(core.ProjectKindUnknown):
		return core.ProjectKindUnknown, nil
	default:
		return "", fmt.Errorf("unknown --kind %q (want auto|sqlc|ent|unknown)", flag)
	}
}

func init() {
	skillsCmd.Flags().StringP("dir", "d", "", "project root (default: nearest go.mod)")
	skillsCmd.Flags().String("name", skills.DefaultName, "skill name and directory")
	skillsCmd.Flags().String("kind", "auto", "project stack: auto|sqlc|ent|unknown")
	skillsCmd.Flags().Bool("print", false, "print the skill to stdout instead of writing it")

	rootCmd.AddCommand(skillsCmd)
}
