package main

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/geodro/lerd/internal/desktopnotify"
	"github.com/geodro/lerd/internal/tray"
	"github.com/geodro/lerd/internal/winshim"
	"github.com/geodro/lerd/internal/wsl"
)

// trayBackend is lerd inside the distro, reached the way `lerd.exe <args>` is.
type trayBackend struct{ cfg winshim.Config }

func (b trayBackend) Command(args ...string) *exec.Cmd {
	cmd := exec.Command("wsl.exe", winshim.WSLArgs(b.cfg, "~", args)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	// The tray shows no output, so what a click did is only visible in the agent log.
	log.Printf("lerd %s", strings.Join(args, " "))
	cmd.Stdout, cmd.Stderr = log.Writer(), log.Writer()
	return cmd
}

func (b trayBackend) Settings() tray.Settings {
	cmd := b.Command("tray", "state")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Run()
	out := buf.Bytes()
	if err != nil {
		log.Printf("reading the tray settings: %v", err)
		return tray.Settings{}
	}
	s, err := tray.ParseSettings(out)
	if err != nil {
		log.Printf("reading the tray settings: %v: %s", err, out)
	}
	return s
}

// Notify raises a Lerd toast for a tray action: silent when it worked, with the
// default sound when it failed. It does not wait for PowerShell.
func (b trayBackend) Notify(title, body string, failed bool) {
	req := desktopnotify.Request{Summary: title, Body: body}
	if failed {
		req.Urgency = desktopnotify.UrgencyNormal
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", wsl.EncodePowerShell(desktopnotify.ToastScript(req)))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	go func() {
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("showing the toast %q: %v: %s", title, err, out)
		}
	}()
}

// runTray shows the tray until quitTray; it must run on the main goroutine.
func runTray(cfg winshim.Config) { tray.RunResident(trayBackend{cfg}) }

func quitTray() { tray.Quit() }
