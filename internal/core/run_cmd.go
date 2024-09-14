package core

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
	"syscall"
)

func RunWithDir(name string, dir string, env []string, args ...string) (string, error) {
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

// RunAsync run command async
func RunAsync(name string, args ...string) error {
	runCmd := exec.Command(name, args...)
	stdOutPipe, _ := runCmd.StdoutPipe()
	stdErrPipe, _ := runCmd.StderrPipe()
	if err := runCmd.Start(); err != nil {
		return fmt.Errorf("failed to start cmd: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go write(ctx, cancel, stdOutPipe)
	go write(ctx, cancel, stdErrPipe)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("quit")
		cancel()
		_ = runCmd.Process.Kill()
		return nil
	case <-ctx.Done():
		log.Println("done")
		return nil
	}
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
