package proc

import (
	"os"
	"os/exec"
	"strings"
)

// Process is the struct for a single process.
type Process struct {
	exec.Cmd
	PPid int
}

// NewProcess creates and returns a new Process.
func NewProcess(path string, args []string, environment ...[]string) *Process {
	env := os.Environ()
	if len(environment) > 0 {
		env = append(env, environment[0]...)
	}
	process := &Process{
		PPid: os.Getpid(),
		Cmd: exec.Cmd{
			Args:       []string{path},
			Path:       path,
			Stdin:      os.Stdin,
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
			Env:        env,
			ExtraFiles: make([]*os.File, 0),
		},
	}
	process.Dir, _ = os.Getwd()
	if len(args) > 0 {
		// Exclude of current binary path.
		start := 0
		if strings.EqualFold(path, args[0]) {
			start = 1
		}
		process.Args = append(process.Args, args[start:]...)
	}
	return process
}

// NewProcessCmd creates and returns a process with given command and optional environment variable array.
func NewProcessCmd(cmd string, environment ...[]string) *Process {
	return NewProcess(GetShell(), append([]string{getShellOption()}, parseCommand(cmd)...), environment...)
}
