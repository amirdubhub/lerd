package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Sections of a checkout's uncommitted work, in the order the dashboard lists them.
const (
	SectionStaged    = "staged"
	SectionUnstaged  = "unstaged"
	SectionUntracked = "untracked"
)

// maxPatchBytes keeps one generated or minified file from flooding the browser.
const maxPatchBytes = 512 << 10

// ChangedFile is one path with git's status letter (A, M, D, T, U, or ? for untracked).
type ChangedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// Changes is a checkout's uncommitted work split the way `git status` splits it.
type Changes struct {
	Staged    []ChangedFile `json:"staged"`
	Unstaged  []ChangedFile `json:"unstaged"`
	Untracked []ChangedFile `json:"untracked"`
}

// Patch is one file's unified diff; Truncated means it was cut at maxPatchBytes.
type Patch struct {
	Patch     string `json:"patch"`
	Truncated bool   `json:"truncated"`
}

// ReadChanges lists the files in each section of dir's uncommitted work.
func ReadChanges(dir string) (Changes, error) {
	var c Changes
	var err error
	if c.Staged, err = sectionFiles(dir, SectionStaged); err != nil {
		return Changes{}, err
	}
	if c.Unstaged, err = sectionFiles(dir, SectionUnstaged); err != nil {
		return Changes{}, err
	}
	if c.Untracked, err = sectionFiles(dir, SectionUntracked); err != nil {
		return Changes{}, err
	}
	return c, nil
}

func sectionFiles(dir, section string) ([]ChangedFile, error) {
	switch section {
	case SectionStaged:
		return nameStatus(dir, "diff", "--cached", "--name-status", "--no-renames", "-z")
	case SectionUnstaged:
		return nameStatus(dir, "diff", "--name-status", "--no-renames", "-z")
	case SectionUntracked:
		out, err := Output(dir, "ls-files", "--others", "--exclude-standard", "-z")
		if err != nil {
			return nil, err
		}
		files := []ChangedFile{}
		for _, p := range strings.Split(out, "\x00") {
			if p != "" {
				files = append(files, ChangedFile{Path: p, Status: "?"})
			}
		}
		return files, nil
	}
	return nil, fmt.Errorf("unknown diff section %q", section)
}

func nameStatus(dir string, args ...string) ([]ChangedFile, error) {
	out, err := Output(dir, args...)
	if err != nil {
		return nil, err
	}
	files := []ChangedFile{}
	fields := strings.Split(out, "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] != "" {
			files = append(files, ChangedFile{Path: fields[i+1], Status: fields[i][:1]})
		}
	}
	return files, nil
}

// FilePatch is the diff of one file in section, ignoring whitespace changes when
// asked. The path must be one git lists there: an untracked diff reads the file
// as given, so anything else is refused.
func FilePatch(dir, section, path string, ignoreSpace bool) (Patch, error) {
	files, err := sectionFiles(dir, section)
	if err != nil {
		return Patch{}, err
	}
	if !listed(files, path) {
		return Patch{}, fmt.Errorf("%q has no %s changes", path, section)
	}
	// A repo's own diff.external or textconv driver must not run from the dashboard.
	args := []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv", "--no-renames"}
	if ignoreSpace {
		args = append(args, "--ignore-all-space")
	}
	switch section {
	case SectionStaged:
		args = append(args, "--cached", "--", path)
	case SectionUnstaged:
		args = append(args, "--", path)
	case SectionUntracked:
		args = append(args, "--no-index", "--", "/dev/null", path)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	// --no-index exits 1 when the files differ, which for an untracked file they always do.
	var exit *exec.ExitError
	if err != nil && !(section == SectionUntracked && errors.As(err, &exit) && exit.ExitCode() == 1) {
		return Patch{}, fmt.Errorf("git diff %s: %w", path, err)
	}
	if len(out) > maxPatchBytes {
		cut := bytes.LastIndexByte(out[:maxPatchBytes], '\n') + 1
		return Patch{Patch: string(out[:cut]), Truncated: true}, nil
	}
	return Patch{Patch: string(out)}, nil
}

func listed(files []ChangedFile, path string) bool {
	for _, f := range files {
		if f.Path == path {
			return true
		}
	}
	return false
}
