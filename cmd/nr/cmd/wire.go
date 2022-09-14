/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Xwudao/neter/pkg/utils"
	"github.com/spf13/cobra"
)

// wireCmd represents the wire command
var wireCmd = &cobra.Command{
	Use:   "wire",
	Short: "wire the dependency",
	Run: func(cmd *cobra.Command, args []string) {
		base, err := os.Getwd()
		if err != nil {
			base = "."
		}
		newWire(base).wire()
	},
}

type wire struct {
	baseDir string
}

func newWire(baseDir string) *wire {
	return &wire{baseDir: baseDir}
}

func (w *wire) wire() {
	files := utils.LoadFiles(w.baseDir, func(filename string) bool {
		return filepath.Base(filename) == "wire.go"
	})
	if len(files) == 0 {
		log.Println("no wire.go file found")
		return
	}

	for _, file := range files {
		log.Printf("wire.go file found: %s\n", file)
		dir := filepath.Dir(file)
		if res, err := runWithDir("wire", dir, nil, "gen"); err != nil {
			log.Println(res)
			log.Printf("wire gen error: %v\n", err)
			continue
		}
		log.Println("wire gen success")
	}
}

func init() {
	rootCmd.AddCommand(wireCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wireCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wireCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
