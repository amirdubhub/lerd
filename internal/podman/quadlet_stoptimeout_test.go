package podman

import (
	"strconv"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// withStopTimeoutKey pins which of the two spellings the generator emits, so a
// test asserts one shape rather than whichever the host's podman supports.
func withStopTimeoutKey(t *testing.T, supported bool) {
	t.Helper()
	orig := supportsContainerStopTimeoutKey
	t.Cleanup(func() { supportsContainerStopTimeoutKey = orig })
	supportsContainerStopTimeoutKey = func() bool { return supported }
}

// TestGenerateCustomQuadlet_DeclaredStopTimeout is the regression test for the
// data-loss case. A database checkpointing a large buffer pool needs longer
// than the default, and being SIGKILLed part-way through costs a crash recovery
// pass on the next start.
func TestGenerateCustomQuadlet_DeclaredStopTimeout(t *testing.T) {
	withStopTimeoutKey(t, true)
	svc := &config.CustomService{Name: "postgres", Image: "postgres:17", StopTimeout: 60}

	content := GenerateCustomQuadlet(svc)
	if !strings.Contains(content, "StopTimeout=60\n") {
		t.Errorf("declared stop timeout was not honoured\n\n%s", content)
	}
	if strings.Contains(content, "StopTimeout=5\n") {
		t.Error("the default must not survive alongside a declared timeout")
	}
}

// TestGenerateCustomQuadlet_DeclaredStopTimeoutOnOldPodman pins that podman
// <5.0, which aborts on the StopTimeout= key and would leave the install with
// no service units at all (#299), still gets the declared window.
func TestGenerateCustomQuadlet_DeclaredStopTimeoutOnOldPodman(t *testing.T) {
	withStopTimeoutKey(t, false)
	svc := &config.CustomService{Name: "postgres", Image: "postgres:17", StopTimeout: 60}

	content := GenerateCustomQuadlet(svc)
	if !strings.Contains(content, "PodmanArgs=--stop-timeout=60\n") {
		t.Errorf("old-podman fallback dropped the declared timeout\n\n%s", content)
	}
	if strings.Contains(content, "StopTimeout=") {
		t.Error("the StopTimeout= key must stay out of quadlets podman <5.0 reads")
	}
}

// TestGenerateCustomQuadlet_UnitTimeoutOutlastsTheContainer is the other half of
// the fix. podman only starts counting its own window once the stop reaches it,
// so a unit timeout equal to it kills the stop mid-shutdown. Arch-family distros
// ship DefaultTimeoutStopSec=10s, which alone would SIGKILL a 60s database stop
// at 10 seconds however long podman was told to wait.
func TestGenerateCustomQuadlet_UnitTimeoutOutlastsTheContainer(t *testing.T) {
	withStopTimeoutKey(t, true)

	for _, tc := range []struct {
		name     string
		declared int
	}{
		{"default", 0},
		{"database", 60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &config.CustomService{Name: "db", Image: "postgres:17", StopTimeout: tc.declared}
			content := GenerateCustomQuadlet(svc)

			want := "TimeoutStopSec=" + strconv.Itoa(svc.UnitStopTimeoutSecs())
			if !strings.Contains(content, want+"\n") {
				t.Errorf("missing %q\n\n%s", want, content)
			}
			if svc.UnitStopTimeoutSecs() <= svc.StopTimeoutSecs() {
				t.Errorf("unit timeout %d must outlast the container's %d",
					svc.UnitStopTimeoutSecs(), svc.StopTimeoutSecs())
			}
		})
	}
}

// TestGenerateCustomQuadlet_SigtermExitIsAStop pins that a main process with no
// SIGTERM handler, which dies with 143 on every stop, ends the unit inactive
// rather than failed.
func TestGenerateCustomQuadlet_SigtermExitIsAStop(t *testing.T) {
	withStopTimeoutKey(t, true)
	svc := &config.CustomService{Name: "search", Image: "getmeili/meilisearch:v1.42"}

	content := GenerateCustomQuadlet(svc)
	if !strings.Contains(content, "\nSuccessExitStatus=143\n") {
		t.Errorf("a SIGTERM exit must count as a clean stop\n\n%s", content)
	}
}
