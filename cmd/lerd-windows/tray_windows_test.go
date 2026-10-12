package main

import (
	"reflect"
	"testing"

	"github.com/geodro/lerd/internal/winshim"
)

func TestTrayBackend_RunsLerdInTheDistroWithoutAWindow(t *testing.T) {
	b := trayBackend{winshim.Config{Distro: "Ubuntu-24.04", Lerd: "/home/juane/.local/bin/lerd"}}

	cmd := b.Command("service", "start", "mysql")

	want := []string{"-d", "Ubuntu-24.04", "--cd", "~", "--exec", "/home/juane/.local/bin/lerd", "service", "start", "mysql"}
	if !reflect.DeepEqual(cmd.Args[1:], want) {
		t.Errorf("args = %q, want %q", cmd.Args[1:], want)
	}
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Error("a tray click must not flash a console window")
	}
}
