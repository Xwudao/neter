//go:build windows

package cmd

import (
	"os"
	"os/exec"
)

func configureDevCommand(cmd *exec.Cmd) {
}

func interruptDevProcess(cmd *exec.Cmd) error {
	return cmd.Process.Signal(os.Interrupt)
}

func killDevProcess(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}
