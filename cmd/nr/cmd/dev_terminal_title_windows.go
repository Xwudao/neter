//go:build windows

package cmd

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var setConsoleTitleW = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleTitleW")

// setDevTerminalTitle uses the native Windows console API, which works in
// Windows Terminal as well as the classic console host.
func setDevTerminalTitle() {
	projectDir, err := os.Getwd()
	if err != nil {
		return
	}
	title, err := windows.UTF16PtrFromString(devTerminalTitle(projectDir))
	if err != nil {
		return
	}
	setConsoleTitleW.Call(uintptr(unsafe.Pointer(title)))
}
