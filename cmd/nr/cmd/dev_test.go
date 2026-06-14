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
