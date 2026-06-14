//go:build windows

package cmd

import (
	"fmt"
	"os/exec"
	"strconv"
)

func configureDevCommand(cmd *exec.Cmd) {
}

func interruptDevProcess(cmd *exec.Cmd) error {
	if err := taskkill(cmd, false); err != nil {
		if forceErr := taskkill(cmd, true); forceErr != nil {
			return fmt.Errorf("graceful stop failed (%v), force stop failed (%v)", err, forceErr)
		}
	}
	return nil
}

func killDevProcess(cmd *exec.Cmd) error {
	return taskkill(cmd, true)
}

func taskkill(cmd *exec.Cmd, force bool) error {
	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	args := []string{"/PID", strconv.Itoa(cmd.Process.Pid), "/T"}
	if force {
		args = append(args, "/F")
	}

	killCmd := exec.Command("taskkill", args...)
	if output, err := killCmd.CombinedOutput(); err != nil {
		if len(output) > 0 {
			return fmt.Errorf("taskkill %v: %w", args, err)
		}
		return fmt.Errorf("taskkill %v: %w", args, err)
	}
	return nil
}
