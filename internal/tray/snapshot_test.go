//go:build !nogui

package tray

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

// stubAPI stands in for lerd-ui. Handlers are keyed by path; a path with no
// handler answers 500, which is how a partial poll is reproduced.
func stubAPI(t *testing.T, handlers map[string]http.HandlerFunc) {
	t.Helper()
	isolateState(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h, ok := handlers[r.URL.Path]; ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	prev := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = prev })
	useBackend(t, &fakeBackend{})
}

// fakeBackend records the commands the tray runs and serves fixed settings.
type fakeBackend struct {
	settings      Settings
	settingsReads int
	ran           [][]string
}

func (f *fakeBackend) Settings() Settings {
	f.settingsReads++
	return f.settings
}

func (f *fakeBackend) Command(args ...string) *exec.Cmd {
	f.ran = append(f.ran, args)
	return exec.Command("lerd-test-command-that-does-not-exist")
}

func useBackend(t *testing.T, b Backend) {
	t.Helper()
	prev := backend
	backend = b
	t.Cleanup(func() { backend = prev })
}

func TestFetchSnapshot_ReadsSettingsFromTheBackend(t *testing.T) {
	stubAPI(t, map[string]http.HandlerFunc{"/api/status": writeJSON(runningStatus)})
	want := Settings{Autostart: true, LANExposed: true, LANServicesExposed: true, Dumps: true, Notifications: true, HighContrastIcon: true, LatestVersion: "v9.9.9"}
	useBackend(t, &fakeBackend{settings: want})

	snap := fetchSnapshot()

	got := Settings{
		Autostart: snap.AutostartEnabled, LANExposed: snap.LANExposed, LANServicesExposed: snap.LANServicesExposed,
		Dumps: snap.DumpsEnabled, Notifications: snap.NotificationsEnabled, HighContrastIcon: snap.HighContrastIcon,
		LatestVersion: snap.LatestVersion,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// With lerd down there is nothing to toggle, and on Windows each read is a
// trip into the WSL distro.
func TestFetchSnapshot_SkipsSettingsWhileLerdIsDown(t *testing.T) {
	stubAPI(t, nil)
	apiBase = "http://127.0.0.1:1"
	fb := &fakeBackend{}
	useBackend(t, fb)

	fetchSnapshot()

	if fb.settingsReads != 0 {
		t.Errorf("settings read %d times while the API was unreachable", fb.settingsReads)
	}
}

func TestLerdCmd_RunsThroughTheBackend(t *testing.T) {
	fb := &fakeBackend{}
	useBackend(t, fb)

	lerdCmd("service", "start", "mysql")

	if len(fb.ran) != 1 || strings.Join(fb.ran[0], " ") != "service start mysql" {
		t.Errorf("ran %q", fb.ran)
	}
}

// isolateState points every lerd directory at a throwaway one. fetchSnapshot
// reads the global config and stamps the update check on its way through, so
// without this the suite writes into the state of the machine running it.
func isolateState(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
}

func writeJSON(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

const runningStatus = `{"nginx":{"running":true},"dns":{"ok":true,"enabled":true},"php_default":"8.4"}`

func TestFetchSnapshot_ServicesAreKnownWhenTheCallSucceeds(t *testing.T) {
	stubAPI(t, map[string]http.HandlerFunc{
		"/api/status":   writeJSON(runningStatus),
		"/api/services": writeJSON(`[{"name":"mysql","status":"active"}]`),
	})

	snap := fetchSnapshot()

	if !snap.ServicesKnown {
		t.Error("a services call that answered must be marked known")
	}
	if len(snap.Services) != 1 {
		t.Errorf("read %d services, want 1", len(snap.Services))
	}
}

func TestFetchSnapshot_AnEmptyListIsStillAnAnswer(t *testing.T) {
	stubAPI(t, map[string]http.HandlerFunc{
		"/api/status":   writeJSON(runningStatus),
		"/api/services": writeJSON(`[]`),
	})

	snap := fetchSnapshot()

	if !snap.ServicesKnown {
		t.Error("an empty list is an answer, not a failed read")
	}
	if len(snap.Services) != 0 {
		t.Errorf("read %d services, want none", len(snap.Services))
	}
}

// The one this exists for: the services endpoint is the slow one, so a poll
// where it alone fails must not report an install with no services at all.
func TestFetchSnapshot_ServicesAreUnknownWhenOnlyThatCallFails(t *testing.T) {
	stubAPI(t, map[string]http.HandlerFunc{
		"/api/status": writeJSON(runningStatus),
	})

	snap := fetchSnapshot()

	if !snap.Running {
		t.Error("status answered, so the environment is still running")
	}
	if snap.ServicesKnown {
		t.Error("a services call that failed must not be reported as an empty install")
	}
}

// A lerd that is down answers nothing, and there genuinely are no services
// running, so the section is known to be empty rather than merely unread.
func TestFetchSnapshot_AnUnreachableAPIKnowsThereAreNoServices(t *testing.T) {
	isolateState(t)
	prev := apiBase
	apiBase = "http://127.0.0.1:1"
	t.Cleanup(func() { apiBase = prev })

	snap := fetchSnapshot()

	if snap.Running {
		t.Error("an unreachable API means the environment is not running")
	}
	if !snap.ServicesKnown {
		t.Error("a stopped lerd has no services, which the menu must be free to show")
	}
}
