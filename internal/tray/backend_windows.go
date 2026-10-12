//go:build !nogui && windows

package tray

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

// defaultBackend is nil: lerd itself runs inside WSL, so the Windows agent
// hands RunResident a backend that reaches into the distro.
func defaultBackend() Backend { return nil }

// hasLANToggle is true: lerd inside WSL is lerd on Linux, which has the toggle.
func hasLANToggle() bool { return true }

func tryLock(f *os.File) error {
	var ol windows.Overlapped
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol)
}

func setDetachAttrs(*exec.Cmd) {}
