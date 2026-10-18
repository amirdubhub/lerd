package config

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// WorktreeDBEntry tracks one isolated worktree database so lerd can drop it
// later — even after `git worktree remove` deletes the worktree's .lerd.yaml.
type WorktreeDBEntry struct {
	Site    string `yaml:"site"`
	Branch  string `yaml:"branch"`
	Service string `yaml:"service"`
	DBName  string `yaml:"db_name"`
	// Provider is the plugin that made the database. Host, Port, Username and
	// Password come from its answer. Password is nil when the answer omitted it,
	// and a pointer to "" when the plugin set an empty password on purpose.
	Provider string  `yaml:"provider,omitempty"`
	Host     string  `yaml:"host,omitempty"`
	Port     string  `yaml:"port,omitempty"`
	Username string  `yaml:"username,omitempty"`
	Password *string `yaml:"password,omitempty"`
}

type worktreeDBRegistry struct {
	Entries []WorktreeDBEntry `yaml:"worktree_dbs"`
}

var worktreeDBRegistryMu sync.Mutex

// WorktreeDBRegistryPath returns the path to the per-user worktree DB registry.
func WorktreeDBRegistryPath() string {
	return filepath.Join(DataDir(), "worktree-dbs.yaml")
}

// LoadWorktreeDBRegistry returns the persisted entries, an empty slice if the
// file is missing.
func LoadWorktreeDBRegistry() ([]WorktreeDBEntry, error) {
	worktreeDBRegistryMu.Lock()
	defer worktreeDBRegistryMu.Unlock()
	return loadWorktreeDBRegistryLocked()
}

func loadWorktreeDBRegistryLocked() ([]WorktreeDBEntry, error) {
	path := WorktreeDBRegistryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var reg worktreeDBRegistry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return reg.Entries, nil
}

func saveWorktreeDBRegistryLocked(entries []WorktreeDBEntry) error {
	path := WorktreeDBRegistryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var data []byte
	if len(entries) == 0 {
		data = []byte("worktree_dbs: []\n")
	} else {
		var err error
		data, err = yaml.Marshal(worktreeDBRegistry{Entries: entries})
		if err != nil {
			return err
		}
	}
	guardRealWrite(path)
	// 0600: entries can hold a plugin database password. WriteFile leaves the
	// mode of an existing file alone, so chmod tightens one an older lerd wrote.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

// AddWorktreeDB inserts or updates an entry by (site, branch).
func AddWorktreeDB(e WorktreeDBEntry) error {
	worktreeDBRegistryMu.Lock()
	defer worktreeDBRegistryMu.Unlock()
	entries, err := loadWorktreeDBRegistryLocked()
	if err != nil {
		return err
	}
	for i, existing := range entries {
		if existing.Site == e.Site && existing.Branch == e.Branch {
			entries[i] = e
			return saveWorktreeDBRegistryLocked(entries)
		}
	}
	entries = append(entries, e)
	return saveWorktreeDBRegistryLocked(entries)
}

// RemoveWorktreeDB deletes the entry for (site, branch) and returns the
// removed entry along with whether one was found.
func RemoveWorktreeDB(site, branch string) (WorktreeDBEntry, bool, error) {
	worktreeDBRegistryMu.Lock()
	defer worktreeDBRegistryMu.Unlock()
	entries, err := loadWorktreeDBRegistryLocked()
	if err != nil {
		return WorktreeDBEntry{}, false, err
	}
	var removed WorktreeDBEntry
	found := false
	kept := entries[:0]
	for _, e := range entries {
		if !found && e.Site == site && e.Branch == branch {
			removed = e
			found = true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		return WorktreeDBEntry{}, false, nil
	}
	return removed, true, saveWorktreeDBRegistryLocked(kept)
}

// FindWorktreeDB returns the entry for (site, branch), if present.
func FindWorktreeDB(site, branch string) (WorktreeDBEntry, bool, error) {
	entries, err := LoadWorktreeDBRegistry()
	if err != nil {
		return WorktreeDBEntry{}, false, err
	}
	for _, e := range entries {
		if e.Site == site && e.Branch == branch {
			return e, true, nil
		}
	}
	return WorktreeDBEntry{}, false, nil
}

// WorktreeDBsForSite returns all entries belonging to a given site.
func WorktreeDBsForSite(site string) ([]WorktreeDBEntry, error) {
	entries, err := LoadWorktreeDBRegistry()
	if err != nil {
		return nil, err
	}
	var out []WorktreeDBEntry
	for _, e := range entries {
		if e.Site == site {
			out = append(out, e)
		}
	}
	return out, nil
}
