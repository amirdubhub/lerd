//go:build linux

package cli

import (
	"errors"

	"github.com/geodro/lerd/internal/platform"
)

func openBrowser(url string) error {
	cmd := platform.OpenCommand(url)
	if cmd == nil {
		return errors.New("no browser opener found; install xdg-utils")
	}
	return cmd.Start()
}
