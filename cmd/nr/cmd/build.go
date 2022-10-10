/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build the final binary",
	Run: func(cmd *cobra.Command, args []string) {
		name := cmd.Flag("name").Value.String()
		arch := cmd.Flag("arch").Value.String()
		dir, _ := cmd.Flags().GetString("dir")
		linux, _ := cmd.Flags().GetBool("linux")
		mac, _ := cmd.Flags().GetBool("mac")
		win, _ := cmd.Flags().GetBool("win")
		// output, _ := cmd.Flags().GetString("output")

		log.SetPrefix("[build] ")
		var (
			res string
			err error
		)
		cmdPath, err := find("cmd")
		if err != nil {
			log.Fatalf(err.Error())
			return
		}
		var appRoot string
		switch len(cmdPath) {
		case 0:
			log.Fatalf("please run in root project directory")
		case 1:
			for _, v := range cmdPath {
				appRoot = v
			}
		default:
			var cmdPaths []string
			for k := range cmdPath {
				cmdPaths = append(cmdPaths, k)
			}
			prompt := &survey.Select{
				Message:  "Which directory do you want to run?",
				Options:  cmdPaths,
				PageSize: 10,
			}
			e := survey.AskOne(prompt, &dir)
			if e != nil || dir == "" {
				return
			}
			appRoot = cmdPath[dir]
		}
		var buildPath = fmt.Sprintf("./%s/", appRoot)

		var Config = []struct {
			Name  string
			Type  string
			Build bool
			Env   []string
		}{
			{Name: name + "-linux", Type: "linux", Build: linux, Env: []string{"GOOS=linux", "GOARCH=" + arch}},
			{Name: name + "-mac", Type: "mac", Build: mac, Env: []string{"GOOS=darwin", "GOARCH=" + arch}},
			{Name: name + "-win" + ".exe", Type: "win", Build: win, Env: []string{"GOOS=windows", "GOARCH=" + arch}},
		}

		for _, c := range Config {
			if c.Build {
				// generate app
				log.Println(fmt.Sprintf("building [%s] app", c.Type))
				// var buildStr = fmt.Sprintf(`build -trimpath -ldflags "-s -w -extldflags '-static'" -o %s %s`, c.Name, buildPath)
				// buildArgs, err := windows.DecomposeCommandLine(buildStr)
				var buildArgs = []string{"build", "-trimpath", `-ldflags=-s -w -extldflags '-static'`, "-o", c.Name}
				buildArgs = append(buildArgs, buildPath)
				fmt.Println(buildArgs)

				if err != nil {
					log.Fatalf(err.Error())
					return
				}
				log.Println("build args: ", strings.Join(buildArgs, " "))

				if res, err = runEnv("go", c.Env, buildArgs...); err != nil {
					log.Println(res)
					log.Fatalf("go build error: %v", err)
					return
				}
				log.Println(res)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	buildCmd.Flags().BoolP("linux", "l", false, "build the linux binary")
	buildCmd.Flags().BoolP("win", "w", false, "build the win binary")
	buildCmd.Flags().BoolP("mac", "m", false, "build the mac binary")
	buildCmd.Flags().String("dir", "app", "the directory of the application")
	buildCmd.Flags().StringP("arch", "a", "amd64", "the architecture of the binary")
	buildCmd.Flags().StringP("name", "n", "main", "the generated app name")
	buildCmd.Flags().StringP("output", "o", "", "the output filename")
}
