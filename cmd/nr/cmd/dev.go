package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/core"
	"github.com/Xwudao/neter/pkg/parser"
)

const (
	devColorReset      = "\033[0m"
	devColorBlue       = "\033[34m"
	devColorMagenta    = "\033[35m"
	devScannerBuffer   = 1024 * 1024
	devStopTimeout     = 5 * time.Second
	devBackendProcess  = "backend"
	devFrontendProcess = "frontend"
	devRestartAction   = "restart"
	devStatusAction    = "status"
	devHelpAction      = "help"
	devQuitAction      = "quit"
	devAllProcesses    = "all"
	devBackendAlias    = "b"
	devFrontendAlias   = "f"
	devAllAlias        = "a"
	devStatusAlias     = "st"
	devHelpAlias       = "h"
)

type devProcessSpec struct {
	Name    string
	Color   string
	Path    string
	Args    []string
	WorkDir string
}

type devManagedProcess struct {
	spec       devProcessSpec
	cmd        *exec.Cmd
	done       chan struct{}
	stopping   bool
	restarting bool
}

type devExitEvent struct {
	name string
	cmd  *exec.Cmd
	err  error
}

type devSupervisor struct {
	ctx          context.Context
	cancel       context.CancelFunc
	outputMu     sync.Mutex
	processes    map[string]*devManagedProcess
	exits        chan devExitEvent
	commandLines chan string
	shuttingDown bool
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

		supervisor := newDevSupervisor(ctx, []devProcessSpec{
			{Name: devBackendProcess, Color: devColorBlue, Path: backendName, Args: backendArgs},
			{Name: devFrontendProcess, Color: devColorMagenta, Path: pm, Args: frontendArgs, WorkDir: frontendPath},
		})

		if err := supervisor.Run(); err != nil {
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

func newDevSupervisor(parent context.Context, specs []devProcessSpec) *devSupervisor {
	ctx, cancel := context.WithCancel(parent)
	processes := make(map[string]*devManagedProcess, len(specs))
	for _, spec := range specs {
		specCopy := spec
		processes[spec.Name] = &devManagedProcess{spec: specCopy}
	}
	return &devSupervisor{
		ctx:          ctx,
		cancel:       cancel,
		processes:    processes,
		exits:        make(chan devExitEvent, len(specs)*4),
		commandLines: make(chan string, 8),
	}
}

func (s *devSupervisor) Run() error {
	for _, proc := range s.processes {
		if err := s.startProcess(proc.spec.Name); err != nil {
			s.printControllerMessage(fmt.Sprintf("failed to start %s: %v", proc.spec.Name, err))
		}
	}

	s.printControllerMessage("commands: rs backend|b | rs frontend|f | rs all|a | status|st | help|h | quit")
	go s.readCommands(os.Stdin)

	for {
		select {
		case <-s.ctx.Done():
			s.shutdownProcesses()
			if errors.Is(s.ctx.Err(), context.Canceled) {
				return nil
			}
			return s.ctx.Err()
		case line, ok := <-s.commandLines:
			if !ok {
				s.commandLines = nil
				continue
			}
			s.handleCommand(line)
		case exit := <-s.exits:
			s.handleExit(exit)
			if s.shuttingDown {
				return nil
			}
		}
	}
}

func (s *devSupervisor) startProcess(name string) error {
	proc, ok := s.processes[name]
	if !ok {
		return fmt.Errorf("unknown process %q", name)
	}

	cmd := exec.CommandContext(s.ctx, proc.spec.Path, proc.spec.Args...)
	cmd.Dir = proc.spec.WorkDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("prepare stdout for %s: %w", name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("prepare stderr for %s: %w", name, err)
	}

	go streamDevOutput(stdout, os.Stdout, proc.spec.Name, proc.spec.Color, &s.outputMu)
	go streamDevOutput(stderr, os.Stderr, proc.spec.Name, proc.spec.Color, &s.outputMu)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}

	proc.cmd = cmd
	proc.done = make(chan struct{})
	proc.stopping = false
	proc.restarting = false

	go func(name string, cmd *exec.Cmd, done chan struct{}) {
		err := cmd.Wait()
		s.exits <- devExitEvent{name: name, cmd: cmd, err: err}
		close(done)
	}(name, cmd, proc.done)

	s.printControllerMessage(fmt.Sprintf("started %s", name))
	return nil
}

func (s *devSupervisor) stopProcess(name string, restart bool) error {
	proc, ok := s.processes[name]
	if !ok {
		return fmt.Errorf("unknown process %q", name)
	}
	if proc.cmd == nil {
		proc.restarting = restart
		return nil
	}

	proc.stopping = true
	proc.restarting = restart

	if err := proc.cmd.Process.Signal(os.Interrupt); err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("interrupt %s: %w", name, err)
		}
	}

	select {
	case <-proc.done:
		return nil
	case <-time.After(devStopTimeout):
		s.printControllerMessage(fmt.Sprintf("%s did not exit in time, killing it", name))
		if err := proc.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("kill %s: %w", name, err)
		}
		<-proc.done
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *devSupervisor) restartProcess(target string) error {
	names, err := expandDevCommandTarget(target)
	if err != nil {
		return err
	}
	for _, name := range names {
		if s.isProcessRunning(name) {
			s.printControllerMessage(fmt.Sprintf("restarting %s", name))
		} else {
			s.printControllerMessage(fmt.Sprintf("starting %s", name))
			if err := s.startProcess(name); err != nil {
				s.printControllerMessage(fmt.Sprintf("failed to start %s: %v", name, err))
			}
			continue
		}
		if err := s.stopProcess(name, true); err != nil {
			s.printControllerMessage(fmt.Sprintf("failed to restart %s: %v", name, err))
		}
	}
	return nil
}

func (s *devSupervisor) shutdownProcesses() {
	if s.shuttingDown {
		return
	}
	s.shuttingDown = true
	for _, name := range []string{devBackendProcess, devFrontendProcess} {
		_ = s.stopProcess(name, false)
	}
}

func (s *devSupervisor) handleExit(exit devExitEvent) {
	proc, ok := s.processes[exit.name]
	if !ok {
		return
	}
	if proc.cmd != exit.cmd {
		return
	}

	plannedStop := proc.stopping
	shouldRestart := proc.restarting
	proc.cmd = nil
	proc.done = nil
	proc.stopping = false
	proc.restarting = false

	if s.shuttingDown {
		return
	}
	if shouldRestart {
		if err := s.startProcess(exit.name); err != nil {
			s.printControllerMessage(fmt.Sprintf("failed to restart %s: %v", exit.name, err))
		}
		return
	}
	if plannedStop {
		return
	}
	if exit.err != nil {
		s.printControllerMessage(fmt.Sprintf("%s exited unexpectedly: %v", exit.name, exit.err))
		return
	}
	s.printControllerMessage(fmt.Sprintf("%s exited unexpectedly", exit.name))
}

func (s *devSupervisor) handleCommand(line string) {
	action, target, ok := parseDevControlCommand(line)
	if !ok {
		if strings.TrimSpace(line) != "" {
			s.printControllerMessage("unknown command, use: rs backend|b | rs frontend|f | rs all|a | status|st | help|h | quit")
		}
		return
	}

	switch action {
	case devRestartAction:
		if err := s.restartProcess(target); err != nil {
			s.printControllerMessage(err.Error())
		}
	case devStatusAction:
		s.printStatus()
	case devHelpAction:
		s.printControllerMessage("commands: rs backend|b | rs frontend|f | rs all|a | status|st | help|h | quit")
	case devQuitAction:
		s.cancel()
	}
}

func (s *devSupervisor) printStatus() {
	for _, name := range []string{devBackendProcess, devFrontendProcess} {
		status := s.processStatus(name)
		s.printControllerMessage(fmt.Sprintf("%s: %s", name, status))
	}
}

func (s *devSupervisor) processStatus(name string) string {
	if s.isProcessRunning(name) {
		return "running"
	}
	return "stopped"
}

func (s *devSupervisor) isProcessRunning(name string) bool {
	proc, ok := s.processes[name]
	return ok && proc.cmd != nil
}

func (s *devSupervisor) readCommands(src io.Reader) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return
		case s.commandLines <- scanner.Text():
		}
	}
	close(s.commandLines)
}

func (s *devSupervisor) printControllerMessage(line string) {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	fmt.Fprintln(os.Stdout, formatDevOutputLine("dev", devColorReset, line))
}

func expandDevCommandTarget(target string) ([]string, error) {
	switch target {
	case devBackendProcess, devBackendAlias:
		return []string{devBackendProcess}, nil
	case devFrontendProcess, devFrontendAlias:
		return []string{devFrontendProcess}, nil
	case devAllProcesses, devAllAlias:
		return []string{devBackendProcess, devFrontendProcess}, nil
	default:
		return nil, fmt.Errorf("unknown process %q", target)
	}
}

func parseDevControlCommand(line string) (action string, target string, ok bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return "", "", false
	}

	switch fields[0] {
	case "rs", "restart":
		if len(fields) != 2 {
			return "", "", false
		}
		return devRestartAction, fields[1], true
	case "status":
		return devStatusAction, "", true
	case devStatusAlias:
		return devStatusAction, "", true
	case "help", devHelpAlias:
		return devHelpAction, "", true
	case "q", "quit", "exit":
		return devQuitAction, "", true
	default:
		return "", "", false
	}
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
