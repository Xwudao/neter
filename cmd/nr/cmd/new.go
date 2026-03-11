package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "new ent schema",
	Long:  `new ent schema`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("eg: new User Group ")
			return
		}

		log.SetPrefix("[new]")

		n := NewEntSchema()

		n.New(args)
	},
}

type EntSchema struct {
}

func NewEntSchema() *EntSchema {
	return &EntSchema{}
}

func (n *EntSchema) New(schemas []string) {
	schemaDir, err := findInternalDataDir()
	if err != nil {
		log.Println(err.Error())
		return
	}

	log.Println(schemaDir)
	var args []string
	args = append(args, "new")
	args = append(args, schemas...)
	text, err := core.RunWithDir("ent", schemaDir, nil, args...)
	fmt.Println(text)
	checkErr(err)
}

func init() {
	rootCmd.AddCommand(newCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// newCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// newCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
