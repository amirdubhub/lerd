package tray

import (
	"encoding/json"
	"testing"
)

// lerd.exe on Windows and lerd inside WSL can be different builds, so the JSON
// `lerd tray state` prints must keep its field names.
func TestSettings_JSONFieldNames(t *testing.T) {
	b, err := json.Marshal(Settings{
		Autostart: true, LANExposed: true, LANServicesExposed: true,
		Dumps: true, Notifications: true, HighContrastIcon: true, LatestVersion: "v1.2.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"autostart":true,"lan_exposed":true,"lan_services_exposed":true,"dumps":true,"notifications":true,"high_contrast_icon":true,"latest_version":"v1.2.0"}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

func TestParseSettings(t *testing.T) {
	s, err := ParseSettings([]byte(`{"lan_exposed":true,"notifications":true,"latest_version":"v2.0.0","unknown":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Settings{LANExposed: true, Notifications: true, LatestVersion: "v2.0.0"}); s != want {
		t.Errorf("got %+v, want %+v", s, want)
	}
	if _, err := ParseSettings([]byte("lerd: not found")); err == nil {
		t.Error("output that is not JSON must be an error")
	}
}
