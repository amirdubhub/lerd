package tray

import (
	"encoding/json"
	"os/exec"
)

// Settings are the persisted lerd toggles the tray menu shows. `lerd tray
// state` prints them as JSON for a tray that cannot read them itself, the
// Windows side of a WSL install, so the field names are a contract.
type Settings struct {
	Autostart          bool   `json:"autostart"`
	LANExposed         bool   `json:"lan_exposed"`
	LANServicesExposed bool   `json:"lan_services_exposed"`
	Dumps              bool   `json:"dumps"`
	Notifications      bool   `json:"notifications"`
	HighContrastIcon   bool   `json:"high_contrast_icon"`
	LatestVersion      string `json:"latest_version,omitempty"` // e.g. "v1.2.0" when an update is available
}

// ParseSettings reads the output of `lerd tray state`.
func ParseSettings(b []byte) (Settings, error) {
	var s Settings
	err := json.Unmarshal(b, &s)
	return s, err
}

// Backend is where the tray reads lerd's settings and runs lerd commands: this
// machine, or a WSL distro seen from Windows.
type Backend interface {
	Settings() Settings
	Command(args ...string) *exec.Cmd
}

// Notifier is a Backend that reports how each menu action went, as the Windows
// tray does with toasts.
type Notifier interface {
	Notify(title, body string, failed bool)
}
