/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/parser"
	"github.com/Xwudao/neter/pkg/proc"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run the application by command",
	Long:  `With this command, you can only run with very short string to start the app.`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		win := runtime.GOOS == "windows"
		name := cmd.Flag("name").Value.String()
		del, _ := cmd.Flags().GetBool("delete")
		rmBuild, _ := cmd.Flags().GetBool("rm")
		wire, _ := cmd.Flags().GetBool("wire")
		dir, _ := cmd.Flags().GetString("dir")
		extraCmd, _ := cmd.Flags().GetString("cmd")
		web, _ := cmd.Flags().GetBool("web")
		pm, _ := cmd.Flags().GetString("pm")

		if win {
			name += ".exe"
		}

		var (
			res string
			err error
		)
		appRoot, err := resolveAppRoot(dir, cmd.Flags().Changed("dir"), "run")
		if err != nil {
			log.Fatalf("[run] %v", err)
			return
		}
		if appRoot == "" {
			return
		}
		var buildPath = fmt.Sprintf("./%s/", normalizeCommandPath(appRoot))
		logCommandStep("run", "app=%s binary=%s", appRoot, name)

		// generate wire
		if wire {
			logCommandStep("run", "generating wire")
			if res, err = core.RunWithDir("wire", buildPath, nil, "gen"); err != nil {
				logCommandOutput("run", "wire output", res)
				log.Fatalf("[run] wire generation failed: %v", err)
				return
			}
			logCommandOutput("run", "wire output", res)
			logCommandSuccess("run", "wire generated")
		}

		// remove `build` directory
		if rmBuild {
			err := os.RemoveAll("build")
			if err != nil {
				log.Fatalf("[run] remove build/ failed: %s", err.Error())
				return
			}
			logCommandSuccess("run", "removed build/")
		}

		// build web
		if web {
			checkErr(buildWebAssets(pm))
		}

		// generate app
		logCommandStep("run", "building binary from %s", buildPath)
		var buildArgs = []string{"build", "-o", name}
		buildArgs = append(buildArgs, buildPath)
		if res, err = core.RunWithDir("go", "", nil, buildArgs...); err != nil {
			logCommandOutput("run", "go build output", res)
			log.Fatalf("[run] go build failed: %v", err)
			return
		}
		logCommandOutput("run", "go build output", res)
		logCommandSuccess("run", "built %s", name)

		// run generate app's command
		var innerArgs []string
		if extraCmd != "" {
			innerArgs = append(innerArgs, parser.GetArgs(extraCmd)...)
		}
		if len(innerArgs) > 0 {
			logCommandStep("run", "extra args: %v", innerArgs)
		}

		defer func() {
			if del {
				time.Sleep(time.Millisecond * 500)
				err := os.Remove(name)
				if err != nil {
					log.Fatalf("[run] delete generated binary failed: %s", err.Error())
					return
				}
				logCommandSuccess("run", "removed generated binary %s", name)
			}
		}()

		// just run app
		appPath := proc.SearchBinary(name)
		logCommandStep("run", "starting %s", appPath)
		err = core.RunAsync(appPath, append(args, innerArgs...)...)
		if err != nil {
			log.Fatalf("[run] start failed: %v", err)
			return
		}
		logCommandSummary("run", start, "process finished")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	runCmd.Flags().String("dir", "app", "the directory of the application")
	runCmd.Flags().StringP("cmd", "c", "", "the extra args set to the application")
	runCmd.Flags().StringP("name", "n", "main", "the generated app name")
	runCmd.Flags().BoolP("wire", "w", false, "generate wire file")
	runCmd.Flags().BoolP("rm", "r", false, "remove build/ directory")
	runCmd.Flags().BoolP("delete", "d", false, "delete the generated app")

	runCmd.Flags().Bool("web", false, "build with web assets")
	runCmd.Flags().String("pm", "pnpm", "the package manger")
}
