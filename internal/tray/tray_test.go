//go:build !nogui

package tray

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestOffOn_FlipsState(t *testing.T) {
	if got := offOn(true); got != "off" {
		t.Errorf("offOn(true) = %q, want %q", got, "off")
	}
	if got := offOn(false); got != "on" {
		t.Errorf("offOn(false) = %q, want %q", got, "on")
	}
}

// notifyingBackend is a fakeBackend that also collects the action toasts.
type notifyingBackend struct {
	fakeBackend
	cmd   func() *exec.Cmd
	notes [][2]string
}

func (n *notifyingBackend) Command(args ...string) *exec.Cmd {
	n.fakeBackend.Command(args...)
	return n.cmd()
}

func (n *notifyingBackend) Notify(title, body string, _ bool) {
	n.notes = append(n.notes, [2]string{title, body})
}

// succeeding runs this test binary with no tests selected, which exits 0 anywhere.
func succeeding() *exec.Cmd { return exec.Command(os.Args[0], "-test.run=^$") }

func failing() *exec.Cmd { return exec.Command("lerd-test-command-that-does-not-exist") }

func TestRunAction_RefreshesEvenWhenTheCommandFails(t *testing.T) {
	useBackend(t, &fakeBackend{})
	called := false
	runAction(func() { called = true }, "start")
	if !called {
		t.Error("refresh should be called even if the command fails so the menu still redraws")
	}
}

func TestRunAction_NotifiesSuccess(t *testing.T) {
	nb := &notifyingBackend{cmd: succeeding}
	useBackend(t, nb)
	runAction(func() {}, "service", "stop", "redis")
	if len(nb.notes) != 1 || nb.notes[0][0] != "redis stopped" {
		t.Errorf("notes = %v, want one \"redis stopped\"", nb.notes)
	}
}

func TestRunAction_NotifiesFailureWithTheError(t *testing.T) {
	nb := &notifyingBackend{cmd: failing}
	useBackend(t, nb)
	runAction(func() {}, "service", "start", "redis")
	if len(nb.notes) != 1 || nb.notes[0][0] != "Couldn't start redis" || nb.notes[0][1] == "" {
		t.Errorf("notes = %v, want one \"Couldn't start redis\" with the error", nb.notes)
	}
}

func TestActionMessage(t *testing.T) {
	fail := errors.New("exit status 1")
	cases := []struct {
		args        []string
		err         error
		stderr      string
		title, body string
		show        bool
	}{
		{[]string{"start"}, nil, "", "Lerd started", "", true},
		{[]string{"stop"}, nil, "", "Lerd stopped", "", true},
		{[]string{"quit"}, nil, "", "Lerd stopped", "", true},
		{[]string{"service", "start", "mailpit"}, nil, "", "mailpit started", "", true},
		{[]string{"use", "8.4"}, nil, "", "PHP 8.4 is the default", "", true},
		{[]string{"autostart", "enable"}, nil, "", "Autostart on", "", true},
		{[]string{"lan", "unexpose"}, nil, "", "LAN access off", "", true},
		{[]string{"lan", "services", "on"}, nil, "", "Services on the LAN on", "", true},
		{[]string{"dump", "off"}, nil, "", "Dumps off", "", true},
		{[]string{"notify", "on"}, nil, "", "Notifications on", "", true},
		{[]string{"tray", "icon", "high-contrast"}, nil, "", "Tray icon: high-contrast", "", true},
		{[]string{"dashboard"}, nil, "", "", "", false},
		{[]string{"dashboard"}, fail, "", "Couldn't open the dashboard", "exit status 1", true},
		{[]string{"service", "start", "redis"}, fail, "pulling image\nError: port 6379 in use\n\n", "Couldn't start redis", "Error: port 6379 in use", true},
		{[]string{"frobnicate"}, nil, "", "lerd frobnicate done", "", true},
	}
	for _, c := range cases {
		title, body, show := actionMessage(c.args, c.err, c.stderr)
		if title != c.title || body != c.body || show != c.show {
			t.Errorf("actionMessage(%v, %v) = %q, %q, %v; want %q, %q, %v", c.args, c.err, title, body, show, c.title, c.body, c.show)
		}
	}
}

func TestLerdBin_FallbackWarnsOnceAndReturnsBareName(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	prevLook := lerdBinLookPath
	prevCands := lerdBinCandidates
	prevW := lerdBinWarnW
	lerdBinLookPath = func(string) (string, error) { return "", errors.New("not on PATH") }
	lerdBinCandidates = func() []string { return []string{missing} }
	var buf bytes.Buffer
	lerdBinWarnW = &buf
	lerdBinFallbackWarn = sync.Once{}
	t.Cleanup(func() {
		lerdBinLookPath = prevLook
		lerdBinCandidates = prevCands
		lerdBinWarnW = prevW
		lerdBinFallbackWarn = sync.Once{}
	})

	if got := lerdBin(); got != "lerd" {
		t.Errorf("first call: lerdBin() = %q, want %q (bare fallback)", got, "lerd")
	}
	if !strings.Contains(buf.String(), "could not locate the lerd binary") {
		t.Errorf("expected fallback warning on stderr, got: %q", buf.String())
	}
	first := buf.Len()

	if got := lerdBin(); got != "lerd" {
		t.Errorf("second call: lerdBin() = %q, want %q", got, "lerd")
	}
	if buf.Len() != first {
		t.Errorf("sync.Once should suppress repeat warnings; buffer grew from %d to %d", first, buf.Len())
	}
}

func TestLerdBin_PrefersLookPathHit(t *testing.T) {
	prevLook := lerdBinLookPath
	prevCands := lerdBinCandidates
	lerdBinLookPath = func(string) (string, error) { return "/opt/from-path/lerd", nil }
	lerdBinCandidates = func() []string {
		t.Error("candidates must not be consulted when LookPath succeeds")
		return nil
	}
	t.Cleanup(func() {
		lerdBinLookPath = prevLook
		lerdBinCandidates = prevCands
	})

	if got := lerdBin(); got != "/opt/from-path/lerd" {
		t.Errorf("lerdBin() = %q, want LookPath result", got)
	}
}
