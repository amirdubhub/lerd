//go:build !darwin

package editor

import (
	"os"
	"path/filepath"
	"strings"
)

// installedOffPath finds an editor not on PATH by a desktop entry claiming
// its URL scheme, which is how JetBrains Toolbox and Flatpak install them.
func (e Editor) installedOffPath() bool {
	return schemeHandled(e.url[:strings.Index(e.url, ":")])
}

// dirCommandOffPath runs a Toolbox launcher, which the shell has on PATH but
// the lerd-ui unit does not. The URL scheme addresses a file, not a folder.
func (e Editor) dirCommandOffPath(dir string) []string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	p := filepath.Join(home, ".local/share/JetBrains/Toolbox/scripts", e.bin)
	if st, err := os.Stat(p); err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
		return nil
	}
	return []string{p, dir}
}
