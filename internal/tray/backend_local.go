//go:build !nogui && !windows

package tray

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/geodro/lerd/internal/platform"
)

// localBackend is lerd on this machine.
type localBackend struct{}

func (localBackend) Settings() Settings { return LocalSettings() }

func (localBackend) Command(args ...string) *exec.Cmd { return exec.Command(lerdBin(), args...) }

func defaultBackend() Backend { return localBackend{} }

// hasLANToggle is false on macOS: gvproxy always exposes ports 80/443 on the
// LAN, and only non-privileged ports support IP binding.
func hasLANToggle() bool { return !platform.Current.UsesMachineVM }

func tryLock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func setDetachAttrs(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
