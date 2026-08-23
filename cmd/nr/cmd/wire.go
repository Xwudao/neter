/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/filex"
)

var wireCmd = &cobra.Command{
	Use:   "wire",
	Short: "regenerate dependency injection code for every wire.go below the current directory",
	RunE: func(cmd *cobra.Command, _ []string) error {
		base, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		return newWire(base).wire()
	},
}

type wire struct {
	baseDir string
}

func newWire(baseDir string) *wire {
	return &wire{baseDir: baseDir}
}

// wire regenerates every discovered injector and returns an error if any one
// fails. CLI callers and generators can therefore rely on its exit status.
func (w *wire) wire() error {
	files, err := filex.LoadFiles(w.baseDir, func(filename string) bool {
		return filepath.Base(filename) == "wire.go"
	})
	if err != nil {
		return fmt.Errorf("find wire files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no wire.go file found below %s", w.baseDir)
	}
	sort.Strings(files)

	var errs []error
	for _, file := range files {
		dir := filepath.Dir(file)
		if _, err := core.RunWithDir("wire", dir, nil, "gen"); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", file, err))
			continue
		}
		fmt.Printf("wire generated: %s\n", file)
	}
	if len(errs) > 0 {
		return fmt.Errorf("Wire generation failed: %w", errors.Join(errs...))
	}
	return nil
}

func init() {
	rootCmd.AddCommand(wireCmd)
}
