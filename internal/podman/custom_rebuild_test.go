package podman

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Removing the old image before the build force-removed the running
// container, and systemd spent its restart limit on the missing image, so
// the site stayed down after a rebuild that reported success.
func TestRebuildCustomImage_RestartsBeforeTheOldImageGoes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Containerfile.lerd"), []byte("FROM busybox\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls []string
	ids := []string{"old", "new"}
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, strings.Join(args, " "))
		if args[0] == "image" && args[1] == "inspect" {
			id := ids[0]
			ids = ids[1:]
			return fakeExec(id, "", 0)(name, args...)
		}
		return fakeExec("", "", 0)(name, args...)
	}

	err := RebuildCustomImage("ccapp", dir, nil, func() error {
		calls = append(calls, "restart")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	build := slices.IndexFunc(calls, func(c string) bool { return strings.HasPrefix(c, "build ") })
	restart := slices.Index(calls, "restart")
	rmi := slices.Index(calls, "rmi old")
	if build < 0 || restart < build || rmi < restart {
		t.Fatalf("want build, then restart, then rmi of the old id; calls: %q", calls)
	}
	for _, c := range calls {
		if strings.HasPrefix(c, "rmi -f") {
			t.Errorf("force-removes an image, which kills the container on it: %q", c)
		}
	}
}
