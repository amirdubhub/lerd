//go:build !darwin

package editor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// A JetBrains IDE installed through Toolbox puts its launcher in Toolbox's
// scripts dir, which the shell adds to PATH but the lerd-ui unit never sees.
func TestDirCommandOpensAToolboxScriptOffPath(t *testing.T) {
	isolate(t)
	t.Setenv("PATH", t.TempDir())
	scripts := filepath.Join(os.Getenv("HOME"), ".local/share/JetBrains/Toolbox/scripts")
	if err := os.MkdirAll(scripts, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(scripts, "phpstorm")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeEditorConfig(t, "phpstorm")
	want := []string{exe, "/a"}
	if got := DirCommand("/a"); !reflect.DeepEqual(got, want) {
		t.Fatalf("DirCommand(phpstorm) = %v, want %v", got, want)
	}
}
