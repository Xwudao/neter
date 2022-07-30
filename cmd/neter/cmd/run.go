/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"
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

		if win {
			name += ".exe"
		}
		mainFile := "cmd/app/main.go"
		if _, err := os.Stat(mainFile); err != nil {
			log.Fatalf("please run in root project directory")
			return
		}
		wireFile := "cmd/app/wire_gen.go"

		log.SetPrefix("[run] ")

		if wire {
			log.Println("generating wire...")
			if err := run("wire", "gen", "./cmd/app/"); err != nil {
				log.Fatalf("wire gen error: %v", err)
				return
			}
		}

		log.Println("generating app...")
		if err := run("go", "build", "-o", name, mainFile, wireFile); err != nil {
			log.Fatalf("go build error: %v", err)
			return
		}

		runCmd := exec.Command(name, args...)
		stdOutPipe, _ := runCmd.StdoutPipe()
		stdErrPipe, _ := runCmd.StderrPipe()
		if err := runCmd.Start(); err != nil {
			// handle error
			log.Fatalf("failed to start cmd: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			rd := io.MultiReader(stdOutPipe, stdErrPipe)
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

			//reader := bufio.NewReader(stdOutPipe)
			//line, err := reader.ReadString('\n')
			//for err == nil {
			//	fmt.Println(line)
			//	line, err = reader.ReadString('\n')
			//}
		}()

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

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	err := cmd.Start()
	err = cmd.Wait()

	return err
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

	runCmd.Flags().StringP("name", "n", "main", "the generated app name")
	runCmd.Flags().BoolP("wire", "w", false, "generate wire file")
	runCmd.Flags().BoolP("delete", "d", false, "delete the generated app")
}
