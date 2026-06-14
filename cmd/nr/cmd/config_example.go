package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
)

var configExampleCmd = &cobra.Command{
	Use:     "config-example",
	Aliases: []string{"example"},
	Short:   "print a complete neter.yml example",
	Long:    "Print a complete example of all currently supported neter.yml configuration fields.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(core.ExampleNeterConfigYAML())
	},
}

func init() {
	rootCmd.AddCommand(configExampleCmd)
}
