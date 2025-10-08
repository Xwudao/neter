package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetGitHash fetches the latest Git commit hash
func GetGitHash() (string, error) {
	// Run the git log command to get the latest commit hash
	cmd := exec.Command("git", "log", "-1", "--format=%H")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Execute the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get git commit hash: %v", err)
	}

	// Return the commit hash
	return strings.TrimSpace(out.String()), nil
}
