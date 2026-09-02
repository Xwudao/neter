//go:build !windows

package cmd

import (
	"fmt"
	"os"
)

// setDevTerminalTitle asks terminals that support OSC titles (including Ghostty)
// to show the dev command together with the current project directory.
func setDevTerminalTitle() {
	projectDir, err := os.Getwd()
	if err != nil {
		return
	}
	fmt.Fprintf(os.Stdout, "\033]2;%s\a", devTerminalTitle(projectDir))
}
