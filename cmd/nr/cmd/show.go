package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/pkg/utils"
)

// showCmd represents the gen command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show some by nr",
	Long:  `show something`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("show ent / ")
			return
		}

		arg := args[0]
		log.SetPrefix("[show]")
		s := NewShow()

		switch arg {
		case "ent":
			s.ShowEnt()
		}

	},
}

type Show struct {
}

func NewShow() *Show {
	return &Show{}
}

func (s *Show) ShowEnt() {
	var dir, _ = os.Getwd()
	schemaDir := filepath.Join(dir, "internal/data")
	info, err := os.Stat(schemaDir)
	if err != nil {
		log.Println("please run in project root.")
		return
	}

	if !info.IsDir() {
		utils.CheckErrWithStatus(fmt.Errorf("internal/data dir not exist"))
	}

	log.Println(schemaDir)
	text, err := runWithDir("ent", schemaDir, nil, []string{"describe", "./ent/schema"}...)
	log.Println(text)
	utils.CheckErrWithStatus(err)

}

func init() {
	rootCmd.AddCommand(showCmd)
}
