/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/utils"
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
		output, _ := cmd.Flags().GetString("output")
		dlv, _ := cmd.Flags().GetBool("dlv")
		trim, _ := cmd.Flags().GetBool("trim")
		web, _ := cmd.Flags().GetBool("web")
		pm, _ := cmd.Flags().GetString("pm")
		html, _ := cmd.Flags().GetBool("html")

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
				Message:  "Which directory do you want to build?",
				Options:  cmdPaths,
				PageSize: 10,
			}
			e := survey.AskOne(prompt, &dir)
			if e != nil || dir == "" {
				return
			}
			appRoot = cmdPath[dir]
		}

		if web {
			log.Println("build with web assets")
			b := core.NewBuildWeb(pm)
			err := b.Check()
			utils.CheckErrWithStatus(err)
			err = b.Build()
			utils.CheckErrWithStatus(err)
			err = b.Copy()
			utils.CheckErrWithStatus(err)

			log.Println("build web assets success")
		}

		var buildPath = fmt.Sprintf("./%s/", appRoot)
		if name == "" {
			name = filepath.Base(appRoot)
		}

		var Config = []struct {
			Name  string
			Type  string
			Build bool
			Env   []string
		}{
			{Name: name + "-linux", Type: "linux", Build: linux, Env: []string{"GOOS=linux", "GOARCH=" + arch}},
			{Name: name + "-mac", Type: "mac", Build: mac, Env: []string{"GOOS=darwin", "GOARCH=" + arch}},
			{Name: name + "-win.exe", Type: "win", Build: win, Env: []string{"GOOS=windows", "GOARCH=" + arch}},
		}

		// compute how many binaries to build
		var buildNum int
		for _, c := range Config {
			if c.Build {
				buildNum++
			}
		}

		var buildAppName []string

		for _, c := range Config {
			if c.Build {
				buildAppName = append(buildAppName, c.Name)
				// generate app
				log.Printf("building [%s] app", c.Type)
				// var buildStr = fmt.Sprintf(`build -trimpath -ldflags "-s -w -extldflags '-static'" -o %s %s`, c.Name, buildPath)
				// buildArgs, err := windows.DecomposeCommandLine(buildStr)
				if buildNum == 1 && output != "" {
					c.Name = output
				}
				var buildArgs = []string{"build"}

				var ldflags bytes.Buffer
				ldflags.WriteString(`-ldflags=-s -w -extldflags '-static'`)
				ldflags.WriteString(fmt.Sprintf(` -X 'main.buildTime=%s'`, time.Now().In(utils.CST).Format(time.DateTime)))

				if dlv {
					buildArgs = append(buildArgs, `-gcflags=all=-N -l`)
				} else if trim {
					buildArgs = append(buildArgs, "-trimpath", ldflags.String())
				}
				// var buildArgs = []string{"build", "-trimpath", `-ldflags=-s -w -extldflags '-static'`, "-o", c.Name}
				buildArgs = append(buildArgs, "-o", c.Name)
				buildArgs = append(buildArgs, buildPath)

				if err != nil {
					log.Fatalf(err.Error())
					return
				}
				log.Println("build args: ", strings.Join(buildArgs, " "))

				if res, err = runEnv("go", c.Env, buildArgs...); err != nil {
					log.Println("\n" + res)
					log.Fatalf("go build error: %v", err)
					return
				}
				log.Println("\n" + res)

				if dlv {
					log.Println("now, you can debug with dlv: ")
					log.Printf(`dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./%s`, c.Name)

				}
			}
		}

		if html {
			log.Println("build web / template")
			root, _ := os.Getwd()
			bh := core.NewBuildHtml(root, buildAppName)
			err := bh.Check()
			utils.CheckErrWithStatus(err)
			err = bh.Copy()
			utils.CheckErrWithStatus(err)
			err = bh.Delete()
			utils.CheckErrWithStatus(err)
			log.Println("build web / template success")
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
	buildCmd.Flags().StringP("name", "n", "", "the generated app name")
	buildCmd.Flags().StringP("output", "o", "", "the output filename, this option only works when building one binary")
	buildCmd.Flags().Bool("dlv", false, "generate binary app can be debugged by dlv")
	buildCmd.Flags().Bool("trim", false, "trim the path and other infos")
	buildCmd.Flags().Bool("web", false, "build with web assets")
	buildCmd.Flags().String("pm", "pnpm", "the package manger")
	buildCmd.Flags().Bool("html", false, "build with web / template")
}
