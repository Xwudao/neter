package proc

import (
	"testing"
)

func TestNewProcessCmd(t *testing.T) {
	p := NewProcessCmd("echo hello")
	err := p.Run()
	if err != nil {
		t.Error(err)
	}
}
