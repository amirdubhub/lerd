package main

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/geodro/lerd/internal/tray"
	"github.com/geodro/lerd/internal/winshim"
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

// runTray shows the tray until quitTray; it must run on the main goroutine.
func runTray(cfg winshim.Config) { tray.RunResident(trayBackend{cfg}) }

func quitTray() { tray.Quit() }
