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

		log.SetPrefix("[run] ")

		var (
			res string
			err error
		)
		appRoot, err := resolveAppRoot(dir, cmd.Flags().Changed("dir"), "run")
		if err != nil {
			log.Fatalf("%v", err)
			return
		}
		if appRoot == "" {
			return
		}
		var buildPath = fmt.Sprintf("./%s/", normalizeCommandPath(appRoot))

		// generate wire
		if wire {
			log.Println("generating wire...")
			if res, err = core.RunWithDir("wire", buildPath, nil, "gen"); err != nil {
				log.Println("\n" + res)
				log.Fatalf("wire gen error: %v", err)
				return
			}
			log.Println(res)
		}

		// remove `build` directory
		if rmBuild {
			err := os.RemoveAll("build")
			if err != nil {
				log.Fatalf("delete build/ directory err: %s", err.Error())
				return
			}
		}

		// build web
		if web {
			checkErr(buildWebAssets(pm))
		}

		// generate app
		log.Println("generating app...")
		var buildArgs = []string{"build", "-o", name}
		buildArgs = append(buildArgs, buildPath)
		if res, err = runCommand("go", buildArgs...); err != nil {
			log.Println("\n" + res)
			log.Fatalf("go build error: %v", err)
			return
		}
		log.Println(res)

		// run generate app's command
		var innerArgs []string
		if extraCmd != "" {
			innerArgs = append(innerArgs, parser.GetArgs(extraCmd)...)
		}
		if len(innerArgs) > 0 {
			log.Printf("extra args: %s\n", innerArgs)
		}

		defer func() {
			if del {
				time.Sleep(time.Millisecond * 500)
				err := os.Remove(name)
				if err != nil {
					log.Fatalf("delete generate file err: %s", err.Error())
					return
				}
			}
		}()

		// just run app
		appPath := proc.SearchBinary(name)
		err = core.RunAsync(appPath, append(args, innerArgs...)...)
		if err != nil {
			log.Fatalf("failed to start cmd: %v", err)
			return
		}
		log.Println("done!")
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
