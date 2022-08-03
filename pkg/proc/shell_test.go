package proc

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetShell(t *testing.T) {
	binary := SearchBinary("cmd.exe")
	t.Logf("binary: %s\n", binary)
	if runtime.GOOS == "windows" {
		assert.Equal(t, "C:\\WINDOWS\\system32\\cmd.exe", binary)
	}
}

func TestSearchBinary(t *testing.T) {
	p := SearchBinary("calc.exe")
	t.Logf("p: %s\n", p)
}

func Test_parseCommand(t *testing.T) {
	args := parseCommand(`cmd.exe /c "C:\Users\fsdf" hello`)
	t.Logf("args: %v\n", args)
}
