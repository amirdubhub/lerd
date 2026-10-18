package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorktreeDBRegistry_emptyByDefault(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	entries, err := LoadWorktreeDBRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty registry, got %d entries", len(entries))
	}
}

func TestWorktreeDBRegistry_addAndFind(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	e := WorktreeDBEntry{Site: "acme", Branch: "feat-a", Service: "mysql", DBName: "acme_feat_a"}
	if err := AddWorktreeDB(e); err != nil {
		t.Fatal(err)
	}
	got, found, err := FindWorktreeDB("acme", "feat-a")
	if err != nil || !found {
		t.Fatalf("FindWorktreeDB found=%v err=%v", found, err)
	}
	if got != e {
		t.Errorf("FindWorktreeDB = %+v, want %+v", got, e)
	}
}

func TestWorktreeDBRegistry_addUpdatesExistingEntry(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	if err := AddWorktreeDB(WorktreeDBEntry{Site: "acme", Branch: "feat-a", Service: "mysql", DBName: "old"}); err != nil {
		t.Fatal(err)
	}
	if err := AddWorktreeDB(WorktreeDBEntry{Site: "acme", Branch: "feat-a", Service: "mysql", DBName: "new"}); err != nil {
		t.Fatal(err)
	}
	entries, _ := LoadWorktreeDBRegistry()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after update, got %d", len(entries))
	}
	if entries[0].DBName != "new" {
		t.Errorf("DBName = %q, want \"new\"", entries[0].DBName)
	}
}

func TestWorktreeDBRegistry_removeReturnsEntry(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	e := WorktreeDBEntry{Site: "acme", Branch: "feat-a", Service: "mysql", DBName: "acme_feat_a"}
	if err := AddWorktreeDB(e); err != nil {
		t.Fatal(err)
	}
	got, found, err := RemoveWorktreeDB("acme", "feat-a")
	if err != nil || !found {
		t.Fatalf("RemoveWorktreeDB found=%v err=%v", found, err)
	}
	if got != e {
		t.Errorf("RemoveWorktreeDB returned %+v, want %+v", got, e)
	}
	_, stillFound, _ := FindWorktreeDB("acme", "feat-a")
	if stillFound {
		t.Errorf("entry still in registry after Remove")
	}
}

func TestWorktreeDBRegistry_passwordFileIsPrivate(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path := WorktreeDBRegistryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("worktree_dbs: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	pass := "s3cret"
	if err := AddWorktreeDB(WorktreeDBEntry{Site: "acme", Branch: "feat", Service: "mysql", DBName: "db", Password: &pass}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode = %o, want 0600", info.Mode().Perm())
	}
	got, found, err := FindWorktreeDB("acme", "feat")
	if err != nil || !found || got.Password == nil || *got.Password != "s3cret" {
		t.Fatalf("password round trip = %+v found=%v err=%v", got.Password, found, err)
	}
	empty := ""
	if err := AddWorktreeDB(WorktreeDBEntry{Site: "acme", Branch: "feat", Service: "mysql", DBName: "db", Password: &empty}); err != nil {
		t.Fatal(err)
	}
	got, found, err = FindWorktreeDB("acme", "feat")
	if err != nil || !found || got.Password == nil || *got.Password != "" {
		t.Fatalf("empty password not preserved: %+v found=%v err=%v", got.Password, found, err)
	}
}

func TestWorktreeDBsForSite(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_ = AddWorktreeDB(WorktreeDBEntry{Site: "acme", Branch: "feat-a", DBName: "acme_a"})
	_ = AddWorktreeDB(WorktreeDBEntry{Site: "acme", Branch: "feat-b", DBName: "acme_b"})
	_ = AddWorktreeDB(WorktreeDBEntry{Site: "other", Branch: "feat-a", DBName: "other_a"})

	got, err := WorktreeDBsForSite("acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries for acme, got %d", len(got))
	}
}
