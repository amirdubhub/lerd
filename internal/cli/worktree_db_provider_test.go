package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/plugins"
)

// providerFixture is a worktree sharing acme's MySQL database and a plugin in
// charge of worktree databases running script. It returns the site, the
// worktree checkout and the file the plugin logs its calls to.
func providerFixture(t *testing.T, script string) (*config.Site, string, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	sitePath, checkout := makeFakeWorktree(t, "feature")
	for _, dir := range []string{sitePath, checkout} {
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_HOST=lerd-mysql\nDB_DATABASE=acme\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	orig := restartWorktreeWorkersFn
	restartWorktreeWorkersFn = func(*config.Site, string) error { return nil }
	t.Cleanup(func() { restartWorktreeWorkersFn = orig })

	dir := filepath.Join(plugins.Dir(), "brancher")
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(t.TempDir(), "calls")
	manifest := "api_version: 1\nname: brancher\nprovides:\n  worktree_database: bin/db\n"
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\nread -r req\necho \"$1 $req\" >> " + calls + "\n" + script
	if err := os.WriteFile(filepath.Join(dir, "bin", "db"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return &config.Site{Name: "acme", Path: sitePath, Domains: []string{"acme.test"}}, checkout, calls
}

func providerFixtureManifest(t *testing.T, script, manifest string) (*config.Site, string, string) {
	t.Helper()
	site, checkout, calls := providerFixture(t, script)
	dir := filepath.Join(plugins.Dir(), "brancher")
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return site, checkout, calls
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, _ := os.ReadFile(path)
	return string(data)
}

func TestSetWorktreeDBIsolated_HandsTheDatabaseToThePluginInCharge(t *testing.T) {
	site, checkout, calls := providerFixture(t, `echo '{"host":"lerd-brancher","database":"acme_feature_br"}'`+"\n")

	if err := SetWorktreeDBIsolated(site, "feature", true, "main"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	log := readFile(t, calls)
	if !strings.HasPrefix(log, "create ") || !strings.Contains(log, `"source":"main"`) || !strings.Contains(log, `"database":"acme_feature"`) {
		t.Errorf("the plugin got %q", log)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_HOST=lerd-brancher") || !strings.Contains(env, "DB_DATABASE=acme_feature_br") {
		t.Errorf("worktree env = %q", env)
	}
	entry, ok, _ := config.FindWorktreeDB("acme", "feature")
	if !ok || entry.Provider != "brancher" || entry.DBName != "acme_feature_br" {
		t.Errorf("registry = %+v %v", entry, ok)
	}

	if err := SetWorktreeDBIsolated(site, "feature", false, ""); err != nil {
		t.Fatalf("share again: %v", err)
	}
	if log := readFile(t, calls); !strings.Contains(log, "drop ") || !strings.Contains(log, `"database":"acme_feature_br"`) {
		t.Errorf("the plugin must drop what it made, got %q", log)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_HOST=lerd-mysql") || !strings.Contains(env, "DB_DATABASE=acme\n") {
		t.Errorf("sharing again must restore the parent's host and database, env = %q", env)
	}
}

// A failure leaves the worktree on the shared database, never on a full copy
// lerd makes behind the plugin's back.
func TestSetWorktreeDBIsolated_PluginFailureKeepsTheWorktreeShared(t *testing.T) {
	cases := map[string]string{
		"no space left": "echo 'no space left' >&2\nexit 1\n",
		`"extra"`:       `echo '{"database":"x","extra":true}'` + "\n",
		"database":      `echo '{"host":"lerd-brancher"}'` + "\n",
	}
	for want, script := range cases {
		t.Run(want, func(t *testing.T) {
			site, checkout, _ := providerFixture(t, script)
			err := SetWorktreeDBIsolated(site, "feature", true, "main")
			if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "brancher") {
				t.Fatalf("err = %v, want the plugin and %s named", err, want)
			}
			if env := readFile(t, filepath.Join(checkout, ".env")); env != "DB_HOST=lerd-mysql\nDB_DATABASE=acme\n" {
				t.Errorf("env changed to %q", env)
			}
			if _, ok, _ := config.FindWorktreeDB("acme", "feature"); ok {
				t.Error("nothing may be recorded for a database that was never made")
			}
		})
	}
}

// A database kept from a removed worktree is reconnected to, not replaced by a
// new branch that would leave the kept one behind.
func TestApplyWorktreeDBChoice_ReuseReconnectsToThePluginsDatabase(t *testing.T) {
	site, checkout, calls := providerFixture(t, `
if [ "$1" = ensure ]; then
  echo '{"host":"lerd-brancher","database":"kept_br"}'
  exit 0
fi
echo '{"host":"lerd-brancher","database":"new"}'
`)
	if err := config.AddWorktreeDB(config.WorktreeDBEntry{Site: "acme", Branch: "feature", Service: "mysql", DBName: "kept_br", Host: "lerd-brancher", Provider: "brancher"}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyWorktreeDBChoice(site, "feature", "reuse", nil); err != nil {
		t.Fatal(err)
	}
	if log := readFile(t, calls); !strings.HasPrefix(log, "ensure ") || strings.Contains(log, "create ") {
		t.Errorf("reuse must ask ensure, not create, got %q", log)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_HOST=lerd-brancher") || !strings.Contains(env, "DB_DATABASE=kept_br") {
		t.Errorf("env = %q", env)
	}
}

// A database kept from a removed worktree belongs to whoever made it. Another
// plugin in charge now must not branch a new one over its record.
func TestApplyWorktreeDBChoice_ReuseRefusesADatabaseAnotherPluginMade(t *testing.T) {
	site, checkout, calls := providerFixture(t, `echo '{"database":"acme_feature_br"}'`+"\n")
	kept := config.WorktreeDBEntry{Site: "acme", Branch: "feature", Service: "mysql", DBName: "kept_other", Host: "lerd-other", Provider: "other"}
	if err := config.AddWorktreeDB(kept); err != nil {
		t.Fatal(err)
	}
	err := ApplyWorktreeDBChoice(site, "feature", "reuse", nil)
	if err == nil || !strings.Contains(err.Error(), "other") {
		t.Fatalf("err = %v, want the plugin that made it named", err)
	}
	if log := readFile(t, calls); log != "" {
		t.Errorf("the plugin in charge must not be asked for a new database, got %q", log)
	}
	if got, ok, _ := config.FindWorktreeDB("acme", "feature"); !ok || got != kept {
		t.Errorf("the kept entry must stay as it was, got %+v %v", got, ok)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); env != "DB_HOST=lerd-mysql\nDB_DATABASE=acme\n" {
		t.Errorf("env changed to %q", env)
	}
}

// The same holds when the plugin that made the kept database is gone: lerd
// must not make an empty one of its own and drop the record of the real one.
func TestApplyWorktreeDBChoice_ReuseRefusesADatabaseOfAGonePlugin(t *testing.T) {
	site, _, _ := providerFixture(t, "")
	if err := plugins.Remove("brancher"); err != nil {
		t.Fatal(err)
	}
	kept := config.WorktreeDBEntry{Site: "acme", Branch: "feature", Service: "mysql", DBName: "kept_br", Host: "lerd-brancher", Provider: "brancher"}
	if err := config.AddWorktreeDB(kept); err != nil {
		t.Fatal(err)
	}
	if err := ApplyWorktreeDBChoice(site, "feature", "reuse", nil); err == nil || !strings.Contains(err.Error(), "brancher") {
		t.Fatalf("err = %v, want the gone plugin named", err)
	}
	if got, ok, _ := config.FindWorktreeDB("acme", "feature"); !ok || got != kept {
		t.Errorf("the kept entry must stay as it was, got %+v %v", got, ok)
	}
}

// A database lerd kept before a plugin took charge is still lerd's, so reuse
// reconnects to it rather than asking the plugin for a new one.
func TestApplyWorktreeDBChoice_ReuseReconnectsToLerdsOwnDatabase(t *testing.T) {
	site, checkout, calls := providerFixture(t, `echo '{"database":"acme_feature_br"}'`+"\n")
	kept := config.WorktreeDBEntry{Site: "acme", Branch: "feature", Service: "mysql", DBName: "acme_feature"}
	if err := config.AddWorktreeDB(kept); err != nil {
		t.Fatal(err)
	}
	if err := ApplyWorktreeDBChoice(site, "feature", "reuse", nil); err != nil {
		t.Fatal(err)
	}
	if log := readFile(t, calls); log != "" {
		t.Errorf("the plugin must not be asked for a new database, got %q", log)
	}
	if got, _, _ := config.FindWorktreeDB("acme", "feature"); got != kept {
		t.Errorf("entry = %+v, want lerd's kept one", got)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_DATABASE=acme_feature\n") {
		t.Errorf("env = %q", env)
	}
}

// A drop the plugin fails leaves the worktree on its own database and the
// record in place, so nothing is left behind unaccounted for.
func TestSetWorktreeDBIsolated_FailedDropChangesNothing(t *testing.T) {
	script := "case \"$1\" in drop) echo 'busy' >&2; exit 1;; esac\necho '{\"host\":\"lerd-brancher\",\"database\":\"acme_feature_br\"}'\n"
	site, checkout, _ := providerFixture(t, script)
	if err := SetWorktreeDBIsolated(site, "feature", true, "main"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	before := readFile(t, filepath.Join(checkout, ".env"))

	err := SetWorktreeDBIsolated(site, "feature", false, "")
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("err = %v, want the plugin's error", err)
	}
	if _, ok, _ := config.FindWorktreeDB("acme", "feature"); !ok {
		t.Error("the record must be kept when the drop failed")
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); env != before {
		t.Errorf("env changed to %q", env)
	}
}

// Orphan cleanup forgets a database only once it is actually dropped.
func TestSetWorktreeDBIsolated_PluginMayAnswerPortUserAndPassword(t *testing.T) {
	site, checkout, _ := providerFixture(t, `echo '{"host":"host.containers.internal","database":"acme_feature_br","port":"55432","username":"postgres","password":"s3cret"}'`+"\n")
	parent := "DB_HOST=lerd-mysql\nDB_PORT=3306\nDB_DATABASE=acme\nDB_USERNAME=root\nDB_PASSWORD=lerd\n"
	for _, dir := range []string{site.Path, checkout} {
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(parent), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := SetWorktreeDBIsolated(site, "feature", true, "empty"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	env := readFile(t, filepath.Join(checkout, ".env"))
	for _, want := range []string{"DB_HOST=host.containers.internal", "DB_PORT=55432", "DB_USERNAME=postgres", "DB_PASSWORD=s3cret", "DB_DATABASE=acme_feature_br"} {
		if !strings.Contains(env, want) {
			t.Errorf("env missing %q, got %q", want, env)
		}
	}
	entry, ok, _ := config.FindWorktreeDB("acme", "feature")
	if !ok || entry.Password == nil || *entry.Password != "s3cret" || entry.Port != "55432" {
		t.Errorf("registry = %+v", entry)
	}
	if err := SetWorktreeDBIsolated(site, "feature", false, ""); err != nil {
		t.Fatalf("share: %v", err)
	}
	env = readFile(t, filepath.Join(checkout, ".env"))
	for _, want := range []string{"DB_HOST=lerd-mysql", "DB_PORT=3306", "DB_USERNAME=root", "DB_PASSWORD=lerd", "DB_DATABASE=acme\n"} {
		if !strings.Contains(env, want) {
			t.Errorf("restore missing %q, got %q", want, env)
		}
	}
}

func TestSetWorktreeDBIsolated_EmptyPasswordClearsTheKey(t *testing.T) {
	site, checkout, _ := providerFixture(t, `echo '{"database":"acme_feature_br","password":""}'`+"\n")
	parent := "DB_HOST=lerd-mysql\nDB_DATABASE=acme\nDB_PASSWORD=lerd\n"
	for _, dir := range []string{site.Path, checkout} {
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(parent), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := SetWorktreeDBIsolated(site, "feature", true, "empty"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_PASSWORD=\n") {
		t.Errorf("empty password must be written, env = %q", env)
	}
}

func TestSetWorktreeDBIsolated_OmittedPasswordLeavesTheParentValue(t *testing.T) {
	site, checkout, _ := providerFixture(t, `echo '{"database":"acme_feature_br"}'`+"\n")
	parent := "DB_HOST=lerd-mysql\nDB_DATABASE=acme\nDB_PASSWORD=lerd\n"
	for _, dir := range []string{site.Path, checkout} {
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(parent), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := SetWorktreeDBIsolated(site, "feature", true, "empty"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_PASSWORD=lerd\n") {
		t.Errorf("omitted password must leave the parent value, env = %q", env)
	}
}

func TestSetWorktreeDBIsolated_PluginDeclineFallsBackToLerd(t *testing.T) {
	site, checkout, calls := providerFixture(t, `echo '{"decline":true}'`+"\n")
	origCreate := createDatabaseFn
	t.Cleanup(func() { createDatabaseFn = origCreate })
	var created string
	createDatabaseFn = func(service, name string) (bool, error) {
		created = service + ":" + name
		return true, nil
	}
	if err := SetWorktreeDBIsolated(site, "feature", true, "empty"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	if !strings.Contains(readFile(t, calls), "create ") {
		t.Error("plugin must still be asked")
	}
	if created != "mysql:acme_feature" {
		t.Errorf("lerd create = %q", created)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_DATABASE=acme_feature") || !strings.Contains(env, "DB_HOST=lerd-mysql") {
		t.Errorf("env = %q", env)
	}
	entry, ok, _ := config.FindWorktreeDB("acme", "feature")
	if !ok || entry.Provider != "" {
		t.Errorf("registry = %+v", entry)
	}
}

func TestSetWorktreeDBIsolated_FamiliesOutsideScopeUseLerd(t *testing.T) {
	manifest := "api_version: 1\nname: brancher\nprovides:\n  worktree_database:\n    run: bin/db\n    families: [postgres]\n"
	site, checkout, calls := providerFixtureManifest(t, `echo '{"database":"should-not-run"}'`+"\n", manifest)
	origCreate := createDatabaseFn
	t.Cleanup(func() { createDatabaseFn = origCreate })
	createDatabaseFn = func(string, string) (bool, error) { return true, nil }
	if err := SetWorktreeDBIsolated(site, "feature", true, "empty"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	if log := readFile(t, calls); log != "" {
		t.Errorf("mysql is outside families, plugin got %q", log)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_DATABASE=acme_feature") {
		t.Errorf("env = %q", env)
	}
}

func TestSetWorktreeDBIsolated_ReuseIgnoresNarrowedFamilies(t *testing.T) {
	manifest := "api_version: 1\nname: brancher\nprovides:\n  worktree_database:\n    run: bin/db\n    families: [postgres]\n"
	site, _, calls := providerFixtureManifest(t, `echo '{"host":"lerd-brancher","database":"kept_br"}'`+"\n", manifest)
	origCreate := createDatabaseFn
	t.Cleanup(func() { createDatabaseFn = origCreate })
	createDatabaseFn = func(string, string) (bool, error) {
		t.Fatal("reuse must not create a lerd database")
		return false, nil
	}
	if err := config.AddWorktreeDB(config.WorktreeDBEntry{
		Site: "acme", Branch: "feature", Service: "mysql", DBName: "kept_br", Provider: "brancher",
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetWorktreeDBIsolated(site, "feature", true, "empty"); err != nil {
		t.Fatalf("reuse: %v", err)
	}
	if log := readFile(t, calls); !strings.HasPrefix(log, "ensure ") {
		t.Errorf("calls = %q", log)
	}
	entry, ok, _ := config.FindWorktreeDB("acme", "feature")
	if !ok || entry.Provider != "brancher" || entry.DBName != "kept_br" {
		t.Errorf("registry = %+v", entry)
	}
}

func TestEnsureWorktreeIsolatedDB_AsksPluginToEnsure(t *testing.T) {
	site, checkout, calls := providerFixture(t, `echo '{"host":"host.containers.internal","database":"kept_br","port":"55432"}'`+"\n")
	writeWorktreeLerdYAML(t, checkout, true)
	if err := config.AddWorktreeDB(config.WorktreeDBEntry{
		Site: "acme", Branch: "feature", Service: "mysql", DBName: "kept_br", Provider: "brancher",
	}); err != nil {
		t.Fatal(err)
	}
	created, err := EnsureWorktreeIsolatedDB(site, "feature", checkout)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("ensure must not report a new create")
	}
	if log := readFile(t, calls); !strings.HasPrefix(log, "ensure ") {
		t.Errorf("calls = %q", log)
	}
	if env := readFile(t, filepath.Join(checkout, ".env")); !strings.Contains(env, "DB_HOST=host.containers.internal") || !strings.Contains(env, "DB_PORT=55432") {
		t.Errorf("env = %q", env)
	}
}

func TestDropOrphanedWorktreeDBs_KeepsTheRecordWhenTheDropFails(t *testing.T) {
	site, _, _ := providerFixture(t, "echo 'busy' >&2\nexit 1\n")
	entry := config.WorktreeDBEntry{Site: "acme", Branch: "gone", Service: "mysql", DBName: "acme_gone_br", Provider: "brancher"}
	if err := config.AddWorktreeDB(entry); err != nil {
		t.Fatal(err)
	}
	DropOrphanedWorktreeDBs(site)
	if _, ok, _ := config.FindWorktreeDB("acme", "gone"); !ok {
		t.Error("the record of a database still out there must be kept")
	}
}
