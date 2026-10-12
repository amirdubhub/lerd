//go:build !windows

package main

import "github.com/geodro/lerd/internal/winshim"

// The tray is Windows-only; the package builds elsewhere so it is vetted and
// tested with the rest of lerd.
func runTray(winshim.Config) { select {} }

func quitTray() {}
