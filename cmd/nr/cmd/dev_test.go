package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestBuildDevBackendCommandDefault(t *testing.T) {
	name, args, err := buildDevBackendCommand("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	if name != exe {
		t.Fatalf("expected executable %q, got %q", exe, name)
	}
	if len(args) != 2 || args[0] != "run" || args[1] != "-dr" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildDevBackendCommandCustom(t *testing.T) {
	name, args, err := buildDevBackendCommand("nr run -dr --dir app/admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "nr" {
		t.Fatalf("expected binary nr, got %q", name)
	}
	if len(args) != 4 || args[0] != "run" || args[1] != "-dr" || args[2] != "--dir" || args[3] != "app/admin" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestFormatDevOutputLine(t *testing.T) {
	line := formatDevOutputLine("backend", devColorBlue, "server started")

	if !strings.Contains(line, "[backend]") {
		t.Fatalf("expected backend prefix in %q", line)
	}
	if !strings.Contains(line, "server started") {
		t.Fatalf("expected log content in %q", line)
	}
	if !strings.HasPrefix(line, devColorBlue) {
		t.Fatalf("expected color prefix in %q", line)
	}
	if !strings.Contains(line, devColorReset) {
		t.Fatalf("expected reset code in %q", line)
	}
}

func TestParseDevControlCommand(t *testing.T) {
	action, target, ok := parseDevControlCommand("rs backend")
	if !ok {
		t.Fatal("expected command to be parsed")
	}
	if action != devRestartAction || target != devBackendProcess {
		t.Fatalf("unexpected parse result: action=%q target=%q", action, target)
	}

	action, target, ok = parseDevControlCommand("status")
	if !ok || action != devStatusAction || target != "" {
		t.Fatalf("unexpected status parse: action=%q target=%q ok=%v", action, target, ok)
	}

	action, target, ok = parseDevControlCommand("st")
	if !ok || action != devStatusAction || target != "" {
		t.Fatalf("unexpected st parse: action=%q target=%q ok=%v", action, target, ok)
	}

	action, target, ok = parseDevControlCommand("h")
	if !ok || action != devHelpAction || target != "" {
		t.Fatalf("unexpected help parse: action=%q target=%q ok=%v", action, target, ok)
	}

	action, target, ok = parseDevControlCommand("quit")
	if !ok || action != devQuitAction || target != "" {
		t.Fatalf("unexpected quit parse: action=%q target=%q ok=%v", action, target, ok)
	}
}

func TestParseDevControlCommandInvalid(t *testing.T) {
	cases := []string{
		"",
		"rs",
		"restart",
		"unknown",
	}

	for _, input := range cases {
		if _, _, ok := parseDevControlCommand(input); ok {
			t.Fatalf("expected %q to be invalid", input)
		}
	}
}

func TestExpandDevCommandTarget(t *testing.T) {
	names, err := expandDevCommandTarget(devAllProcesses)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 || names[0] != devBackendProcess || names[1] != devFrontendProcess {
		t.Fatalf("unexpected names: %#v", names)
	}

	names, err = expandDevCommandTarget(devBackendAlias)
	if err != nil {
		t.Fatalf("unexpected backend alias error: %v", err)
	}
	if len(names) != 1 || names[0] != devBackendProcess {
		t.Fatalf("unexpected backend alias names: %#v", names)
	}

	names, err = expandDevCommandTarget(devFrontendAlias)
	if err != nil {
		t.Fatalf("unexpected frontend alias error: %v", err)
	}
	if len(names) != 1 || names[0] != devFrontendProcess {
		t.Fatalf("unexpected frontend alias names: %#v", names)
	}

	names, err = expandDevCommandTarget(devAllAlias)
	if err != nil {
		t.Fatalf("unexpected all alias error: %v", err)
	}
	if len(names) != 2 || names[0] != devBackendProcess || names[1] != devFrontendProcess {
		t.Fatalf("unexpected all alias names: %#v", names)
	}
}

func TestExpandDevCommandTargetInvalid(t *testing.T) {
	if _, err := expandDevCommandTarget("api"); err == nil {
		t.Fatal("expected invalid target error")
	}
}
