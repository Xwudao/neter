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
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/pkg/filex"
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

		if win {
			name += ".exe"
		}
		appRoot := filepath.Join("cmd", dir)
		files := find(dir)
		if len(files) == 0 {
			log.Fatalf("please run in root project directory")
			return
		}
		//mainFile := "cmd/app/main.go"
		//if _, err := os.Stat(mainFile); err != nil {
		//}
		//wireFile := "cmd/app/wire_gen.go"

		log.SetPrefix("[run] ")

		var (
			res string
			err error
		)
		if wire {
			log.Println("generating wire...")
			if res, err = runWithDir("wire", appRoot, "gen"); err != nil {
				log.Println(res)
				log.Fatalf("wire gen error: %v", err)
				return
			}
			log.Println(res)

		}

		log.Println("generating app...")
		var buildArgs = []string{"build", "-o", name}
		buildArgs = append(buildArgs, files...)
		if res, err = run("go", buildArgs...); err != nil {
			log.Println(res)
			log.Fatalf("go build error: %v", err)
			return
		}
		log.Println(res)

		var innerArgs []string
		if len(args) > 1 {
			for _, arg := range args[1:] {
				if strings.Contains(arg, "=") {
					innerArgs = append(innerArgs, "--"+arg)
				} else {
					innerArgs = append(innerArgs, arg)
				}
			}
		}

		//appPath, _ := filepath.Abs(name)
		appPath := proc.SearchBinary(name)

		runCmd := exec.Command(appPath, append(args, innerArgs...)...)
		stdOutPipe, _ := runCmd.StdoutPipe()
		stdErrPipe, _ := runCmd.StderrPipe()
		if err := runCmd.Start(); err != nil {
			// handle error
			log.Fatalf("failed to start cmd: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		go write(ctx, cancel, stdOutPipe)
		go write(ctx, cancel, stdErrPipe)

		quit := make(chan os.Signal)
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
	return runWithDir(name, "", args...)
}
func runWithDir(name string, dir string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
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

func find(dir string) (files []string) {
	fp := filepath.Join("cmd", dir)
	files, err := filex.LoadFiles(fp, func(s string) bool {
		v := filepath.Ext(s) == ".go"
		file, err := os.ReadFile(s)
		if err != nil {
			return false
		}
		return !strings.Contains(string(file), "+build wireinject") && v
	})
	if err == nil {
		return
	}
	return nil
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
	runCmd.Flags().StringP("name", "n", "main", "the generated app name")
	runCmd.Flags().BoolP("wire", "w", false, "generate wire file")
	runCmd.Flags().BoolP("delete", "d", false, "delete the generated app")
}
