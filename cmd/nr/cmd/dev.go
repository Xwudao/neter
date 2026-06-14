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
	"path/filepath"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/parser"
)

const (
	devColorReset    = "\033[0m"
	devColorBlue     = "\033[34m"
	devColorMagenta  = "\033[35m"
	devScannerBuffer = 1024 * 1024
)

type devProcess struct {
	Name  string
	Color string
	Cmd   *exec.Cmd
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "run backend and frontend development commands",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := core.LoadOptionalNeterConfig()
		if err != nil {
			log.Fatalf("load neter.yml error: %v", err)
			return
		}
		devCfg := cfg.EffectiveDevConfig()

		backendCmd, _ := cmd.Flags().GetString("backend-cmd")
		frontendDir, _ := cmd.Flags().GetString("frontend-dir")
		pm, _ := cmd.Flags().GetString("pm")
		frontendCmd, _ := cmd.Flags().GetString("frontend-cmd")

		if !cmd.Flags().Changed("backend-cmd") {
			backendCmd = devCfg.Backend.Cmd
		}
		if !cmd.Flags().Changed("frontend-dir") {
			frontendDir = devCfg.Frontend.Dir
		}
		if !cmd.Flags().Changed("pm") {
			pm = devCfg.Frontend.Pm
		}
		if !cmd.Flags().Changed("frontend-cmd") {
			frontendCmd = devCfg.Frontend.Cmd
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		backendName, backendArgs, err := buildDevBackendCommand(backendCmd)
		if err != nil {
			log.Fatalf("invalid backend command: %v", err)
			return
		}

		frontendPath, err := filepath.Abs(frontendDir)
		if err != nil {
			log.Fatalf("resolve frontend directory: %v", err)
			return
		}
		if info, err := os.Stat(frontendPath); err != nil || !info.IsDir() {
			log.Fatalf("frontend directory not found: %s", frontendPath)
			return
		}

		frontendArgs := parser.GetArgs(frontendCmd)
		if len(frontendArgs) == 0 {
			log.Fatal("frontend command cannot be empty")
			return
		}

		log.Printf("[dev] backend: %s %v", backendName, backendArgs)
		log.Printf("[dev] frontend: %s %v (dir=%s)", pm, frontendArgs, frontendPath)

		backendProc := exec.CommandContext(ctx, backendName, backendArgs...)

		frontendProc := exec.CommandContext(ctx, pm, frontendArgs...)
		frontendProc.Dir = frontendPath

		if err := runDevProcesses(ctx,
			devProcess{Name: "backend", Color: devColorBlue, Cmd: backendProc},
			devProcess{Name: "frontend", Color: devColorMagenta, Cmd: frontendProc},
		); err != nil {
			log.Fatalf("dev command failed: %v", err)
		}
	},
}

func buildDevBackendCommand(backendCmd string) (string, []string, error) {
	if backendCmd == "" {
		exePath, err := os.Executable()
		if err != nil {
			return "", nil, err
		}
		return exePath, []string{"run", "-dr"}, nil
	}

	args := parser.GetArgs(backendCmd)
	if len(args) == 0 {
		return "", nil, fmt.Errorf("empty backend command")
	}
	return args[0], args[1:], nil
}

func runDevProcesses(ctx context.Context, procs ...devProcess) error {
	type result struct {
		name string
		err  error
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan result, len(procs))
	started := make([]*exec.Cmd, 0, len(procs))
	var outputMu sync.Mutex

	for _, proc := range procs {
		stdout, err := proc.Cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("prepare stdout for %s: %w", proc.Name, err)
		}
		stderr, err := proc.Cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("prepare stderr for %s: %w", proc.Name, err)
		}

		go streamDevOutput(stdout, os.Stdout, proc.Name, proc.Color, &outputMu)
		go streamDevOutput(stderr, os.Stderr, proc.Name, proc.Color, &outputMu)

		cmd := proc.Cmd
		if err := cmd.Start(); err != nil {
			cancel()
			for _, startedCmd := range started {
				_ = startedCmd.Wait()
			}
			return fmt.Errorf("start %s: %w", proc.Name, err)
		}
		started = append(started, cmd)

		go func(proc devProcess) {
			results <- result{name: proc.Name, err: proc.Cmd.Wait()}
		}(proc)
	}

	var firstErr error
	for range procs {
		res := <-results
		if res.err != nil && firstErr == nil && ctx.Err() == nil {
			firstErr = fmt.Errorf("%s: %w", res.name, res.err)
			cancel()
		}
		if res.err == nil && firstErr == nil && ctx.Err() == nil {
			firstErr = fmt.Errorf("%s exited", res.name)
			cancel()
		}
	}

	if firstErr != nil {
		return firstErr
	}
	return nil
}

func streamDevOutput(src io.Reader, dst io.Writer, name string, color string, mu *sync.Mutex) {
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), devScannerBuffer)
	for scanner.Scan() {
		mu.Lock()
		fmt.Fprintln(dst, formatDevOutputLine(name, color, scanner.Text()))
		mu.Unlock()
	}
}

func formatDevOutputLine(name string, color string, line string) string {
	return fmt.Sprintf("%s[%s]%s %s", color, name, devColorReset, line)
}

func init() {
	rootCmd.AddCommand(devCmd)

	devCmd.Flags().String("backend-cmd", "", "backend development command, default is current nr binary with `run -dr`")
	devCmd.Flags().String("frontend-dir", "web", "frontend development directory")
	devCmd.Flags().String("pm", "pnpm", "frontend package manager")
	devCmd.Flags().String("frontend-cmd", "run dev", "frontend development command arguments")
}
