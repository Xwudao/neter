package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/utils"
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
	var dir, _ = os.Getwd()
	schemaDir := filepath.Join(dir, "internal/data")
	info, err := os.Stat(schemaDir)
	if err != nil {
		log.Println("please run in project root.")
		return
	}

	if !info.IsDir() {
		utils.CheckErrWithStatus(fmt.Errorf("internal/data directory not exist"))
	}

	log.Println(schemaDir)
	var args []string
	args = append(args, "new")
	args = append(args, schemas...)
	text, err := core.RunWithDir("ent", schemaDir, nil, args...)
	fmt.Println(text)
	utils.CheckErrWithStatus(err)
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
