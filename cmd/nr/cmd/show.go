package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
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
	schemaDir, err := findInternalDataDir()
	if err != nil {
		log.Println(err.Error())
		return
	}

	log.Println(schemaDir)
	text, err := core.RunWithDir("ent", schemaDir, nil, []string{"describe", "./ent/schema"}...)
	fmt.Println(text)
	checkErr(err)

}

func init() {
	rootCmd.AddCommand(showCmd)
}
