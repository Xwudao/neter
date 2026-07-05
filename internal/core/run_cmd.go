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
	"sync"
	"syscall"
)

func RunWithDir(name string, dir string, env []string, args ...string) (string, error) {
	cmd := newCommand(name, dir, env, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// combine stdout + stderr, because some tools (e.g. vite, webpack)
		// output error details to stdout instead of stderr.
		out := stdout.String()
		if errOut := stderr.String(); errOut != "" {
			if out != "" {
				out += "\n"
			}
			out += errOut
		}
		return out, err
	}
	return stdout.String(), nil
}

// RunAsync run command async
func RunAsync(name string, args ...string) error {
	runCmd := newCommand(name, "", nil, args...)
	stdOutPipe, err := runCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("prepare stdout pipe: %w", err)
	}
	stdErrPipe, err := runCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("prepare stderr pipe: %w", err)
	}
	if err := runCmd.Start(); err != nil {
		return fmt.Errorf("failed to start cmd: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once

	go streamLines(ctx, stdOutPipe, func() {
		once.Do(cancel)
	})
	go streamLines(ctx, stdErrPipe, func() {
		once.Do(cancel)
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case <-quit:
		log.Println("quit")
		once.Do(cancel)
		_ = runCmd.Process.Kill()
		return nil
	case <-ctx.Done():
		log.Println("done")
		return nil
	}
}

func newCommand(name string, dir string, env []string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}
	return cmd
}

func streamLines(ctx context.Context, rd io.Reader, onDone func()) {
	defer onDone()

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
		return
	}
}
