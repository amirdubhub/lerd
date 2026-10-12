package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// redirectStderr sends this process's stderr, Go's and the Win32 handle, to f.
func redirectStderr(f *os.File) {
	os.Stderr = f
	windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd())) //nolint:errcheck
}

// disableQuickEdit stops a click inside the installer's console from pausing
// it: in QuickEdit mode Windows freezes any program that writes to a console
// while text is being selected, which looks exactly like a hung install.
func disableQuickEdit() {
	h, err := windows.GetStdHandle(windows.STD_INPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return
	}
	windows.SetConsoleMode(h, (mode&^windows.ENABLE_QUICK_EDIT_MODE)|windows.ENABLE_EXTENDED_FLAGS) //nolint:errcheck
}
