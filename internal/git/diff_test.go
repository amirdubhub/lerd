package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func changesFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	write(t, filepath.Join(dir, "a"), "one\n")
	write(t, filepath.Join(dir, "b"), "two\n")
	write(t, filepath.Join(dir, ".gitignore"), "ignored\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "base")
	write(t, filepath.Join(dir, "a"), "one staged\n")
	gitRun(t, dir, "add", "a")
	write(t, filepath.Join(dir, "a"), "one staged then edited\n")
	gitRun(t, dir, "rm", "-q", "b")
	write(t, filepath.Join(dir, "new file.txt"), "fresh\n")
	write(t, filepath.Join(dir, "ignored"), "secret\n")
	return dir
}

func TestReadChanges(t *testing.T) {
	dir := changesFixture(t)
	got, err := ReadChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	same := func(name string, got, want []ChangedFile) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d]: got %v want %v", name, i, got[i], want[i])
			}
		}
	}
	same("staged", got.Staged, []ChangedFile{{"a", "M"}, {"b", "D"}})
	same("unstaged", got.Unstaged, []ChangedFile{{"a", "M"}})
	same("untracked", got.Untracked, []ChangedFile{{"new file.txt", "?"}})
}

// Each section diffs against its own base: staged against HEAD, unstaged against the index.
func TestFilePatch(t *testing.T) {
	dir := changesFixture(t)
	cases := []struct{ section, path, has, hasNot string }{
		{SectionStaged, "a", "+one staged\n", "edited"},
		{SectionUnstaged, "a", "+one staged then edited", "-one\n"},
		{SectionStaged, "b", "-two", ""},
		{SectionUntracked, "new file.txt", "+fresh", ""},
	}
	for _, c := range cases {
		p, err := FilePatch(dir, c.section, c.path, false)
		if err != nil {
			t.Fatalf("%s %s: %v", c.section, c.path, err)
		}
		if !strings.Contains(p.Patch, c.has) || (c.hasNot != "" && strings.Contains(p.Patch, c.hasNot)) {
			t.Errorf("%s %s: patch\n%s", c.section, c.path, p.Patch)
		}
	}
}

// The untracked diff reads whatever path it gets, so only listed paths may reach it.
func TestFilePatch_refusesUnlistedPath(t *testing.T) {
	dir := changesFixture(t)
	outside := filepath.Join(t.TempDir(), "outside")
	write(t, outside, "x")
	for _, c := range []struct{ section, path string }{
		{SectionUntracked, outside},
		{SectionUntracked, "../outside"},
		{SectionUntracked, "ignored"},
		{SectionUnstaged, "b"},
		{"everything", "a"},
	} {
		if _, err := FilePatch(dir, c.section, c.path, false); err == nil {
			t.Errorf("%s %s: expected a refusal", c.section, c.path)
		}
	}
}

func TestFilePatch_truncatesHugeFiles(t *testing.T) {
	dir := changesFixture(t)
	if err := os.WriteFile(filepath.Join(dir, "huge.js"), []byte(strings.Repeat("x = 1;\n", maxPatchBytes/4)), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := FilePatch(dir, SectionUntracked, "huge.js", false)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Truncated || len(p.Patch) > maxPatchBytes || !strings.HasSuffix(p.Patch, "\n") {
		t.Errorf("truncated=%v len=%d", p.Truncated, len(p.Patch))
	}
}

// The dashboard's own "Initialize git" leaves a repo with no commit yet.
func TestReadChanges_beforeFirstCommit(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	write(t, filepath.Join(dir, "staged.php"), "s\n")
	write(t, filepath.Join(dir, "loose.php"), "l\n")
	gitRun(t, dir, "add", "staged.php")

	got, err := ReadChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Staged) != 1 || got.Staged[0] != (ChangedFile{"staged.php", "A"}) || len(got.Untracked) != 1 {
		t.Fatalf("got %+v", got)
	}
	p, err := FilePatch(dir, SectionStaged, "staged.php", false)
	if err != nil || !strings.Contains(p.Patch, "+s") {
		t.Fatalf("patch %q err %v", p.Patch, err)
	}
}

// Re-indenting a file rewrites every line; ignoring whitespace leaves only the real change.
func TestFilePatch_ignoresWhitespace(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	write(t, filepath.Join(dir, "c.yaml"), "services:\n    - mysql\n    - redis\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "base")
	write(t, filepath.Join(dir, "c.yaml"), "services:\n  - mysql\n  - redis\n  - mailpit\n")

	full, err := FilePatch(dir, SectionUnstaged, "c.yaml", false)
	if err != nil || !strings.Contains(full.Patch, "-    - mysql") {
		t.Fatalf("full: %q %v", full.Patch, err)
	}
	quiet, err := FilePatch(dir, SectionUnstaged, "c.yaml", true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(quiet.Patch, "-    - mysql") || !strings.Contains(quiet.Patch, "+  - mailpit") {
		t.Errorf("ignoring whitespace: %q", quiet.Patch)
	}

	write(t, filepath.Join(dir, "c.yaml"), "services:\n  - mysql\n  - redis\n")
	if p, err := FilePatch(dir, SectionUnstaged, "c.yaml", true); err != nil || strings.Contains(p.Patch, "@@") {
		t.Errorf("whitespace-only change: %q %v", p.Patch, err)
	}
}
