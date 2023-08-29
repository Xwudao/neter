/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

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
		wire, _ := cmd.Flags().GetBool("wire")
		dir, _ := cmd.Flags().GetString("dir")
		extraCmd, _ := cmd.Flags().GetString("cmd")

		if win {
			name += ".exe"
		}

		log.SetPrefix("[run] ")

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

		// generate wire
		if wire {
			log.Println("generating wire...")
			if res, err = runWithDir("wire", buildPath, nil, "gen"); err != nil {
				log.Println("\n" + res)
				log.Fatalf("wire gen error: %v", err)
				return
			}
			log.Println(res)
		}

		// generate app
		log.Println("generating app...")
		var buildArgs = []string{"build", "-o", name}
		buildArgs = append(buildArgs, buildPath)
		if res, err = run("go", buildArgs...); err != nil {
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

		// just run app
		appPath := proc.SearchBinary(name)
		runCmd := exec.Command(appPath, append(args, innerArgs...)...)
		stdOutPipe, _ := runCmd.StdoutPipe()
		stdErrPipe, _ := runCmd.StderrPipe()
		if err := runCmd.Start(); err != nil {
			log.Fatalf("failed to start cmd: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		go write(ctx, cancel, stdOutPipe)
		go write(ctx, cancel, stdErrPipe)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

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

		for {
			select {
			case <-quit:
				log.Println("quit")
				cancel()
				_ = runCmd.Process.Kill()
				return
			case <-ctx.Done():
				log.Println("done")
				return
			}
		}

	},
}

func write(ctx context.Context, cancel context.CancelFunc, rd io.Reader) {
	scanner := bufio.NewScanner(rd)
	scanner.Split(bufio.ScanLines)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		for scanner.Scan() {
			m := scanner.Text()
			fmt.Println(m)
		}
		cancel()
	}
}

func run(name string, args ...string) (string, error) {
	return runWithDir(name, "", nil, args...)
}
func runEnv(name string, env []string, args ...string) (string, error) {
	return runWithDir(name, "", env, args...)
}
func runWithDir(name string, dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}
	return stdout.String(), nil
}

func find(base string) (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(wd, "/") {
		wd += "/"
	}
	var root bool
	next := func(dir string) (map[string]string, error) {
		cmdPath := make(map[string]string)
		err := filepath.Walk(dir, func(walkPath string, info os.FileInfo, err error) error {
			// multi level directory is not allowed under the cmdPath directory, so it is judged that the path ends with cmdPath.
			if strings.HasSuffix(walkPath, "cmd") {
				paths, err := os.ReadDir(walkPath)
				if err != nil {
					return err
				}
				for _, fileInfo := range paths {
					if fileInfo.IsDir() {
						abs := path.Join(walkPath, fileInfo.Name())
						cmdPath[strings.TrimPrefix(abs, wd)] = abs
					}
				}
				return nil
			}
			if info.Name() == "go.mod" {
				root = true
			}
			return nil
		})
		return cmdPath, err
	}
	for i := 0; i < 5; i++ {
		tmp := base
		cmd, err := next(tmp)
		if err != nil {
			return nil, err
		}
		if len(cmd) > 0 {
			return cmd, nil
		}
		if root {
			break
		}
		_ = filepath.Join(base, "..")
	}
	return map[string]string{"": base}, nil
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
	runCmd.Flags().BoolP("delete", "d", false, "delete the generated app")
}
