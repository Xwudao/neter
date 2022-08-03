package proc

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Xwudao/neter/pkg/filex"
)

func GetShell() string {
	switch runtime.GOOS {
	case "windows":
		return SearchBinary("cmd.exe")

	default:
		// Check the default binary storage path.
		if filex.Exists("/bin/bash") {
			return "/bin/bash"
		}
		if filex.Exists("/bin/sh") {
			return "/bin/sh"
		}
		// Else search the env PATH.
		path := SearchBinary("bash")
		if path == "" {
			path = SearchBinary("sh")
		}
		return path
	}
}

// getShellOption returns the shell option depending on current working operating system.
// It returns "/c" for windows, and "-c" for others.
func getShellOption() string {
	switch runtime.GOOS {
	case "windows":
		return "/c"

	default:
		return "-c"
	}
}

// SearchBinary searches the binary `f` in current working folder and PATH environment.
func SearchBinary(f string) string {
	if filex.Exists(f) {
		abs, err := filepath.Abs(f)
		if err != nil {
			return f
		}
		return abs
	}
	return SearchBinaryPath(f)
}

// SearchBinaryPath searches the binary `f` in PATH environment.
func SearchBinaryPath(f string) string {
	array := ([]string)(nil)
	switch runtime.GOOS {
	case "windows":
		path := os.Getenv("PATH")
		array = strings.Split(path, ";")

		if strings.ToLower(filepath.Ext(f)) != ".exe" {
			f += ".exe"
		}

	default:
		path := os.Getenv("PATH")
		array = strings.Split(path, ":")
	}
	if len(array) == 0 {
		return ""
	}
	for _, v := range array {
		if filex.Exists(filepath.Join(v, f)) {
			return filepath.Join(v, f)
		}
	}
	return ""
}

// parseCommand parses command `cmd` into slice arguments.
//
// Note that it just parses the `cmd` for "cmd.exe" binary in windows, but it is not necessary
// parsing the `cmd` for other systems using "bash"/"sh" binary.
func parseCommand(cmd string) (args []string) {
	if runtime.GOOS != "windows" {
		return []string{cmd}
	}
	// Just for "cmd.exe" in windows.
	var argStr string
	var firstChar, prevChar, lastChar1, lastChar2 byte
	array := strings.Split(cmd, " ")
	for _, v := range array {
		if len(argStr) > 0 {
			argStr += " "
		}
		firstChar = v[0]
		lastChar1 = v[len(v)-1]
		lastChar2 = 0
		if len(v) > 1 {
			lastChar2 = v[len(v)-2]
		}
		if prevChar == 0 && (firstChar == '"' || firstChar == '\'') {
			// It should remove the first quote char.
			argStr += v[1:]
			prevChar = firstChar
		} else if prevChar != 0 && lastChar2 != '\\' && lastChar1 == prevChar {
			// It should remove the last quote char.
			argStr += v[:len(v)-1]
			args = append(args, argStr)
			argStr = ""
			prevChar = 0
		} else if len(argStr) > 0 {
			argStr += v
		} else {
			args = append(args, v)
		}
	}
	return
}
