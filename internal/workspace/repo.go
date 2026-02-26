package workspace

import "strings"

// Repo represents a discovered git repository in the workspace.
type Repo struct {
	Alias       string // Directory name, e.g. "my-app-android"
	Path        string // Absolute path to repo root
	Prefix      string // Short branch prefix, e.g. "and"
	Description string // Auto-detected type, e.g. "Android (Kotlin)"
}

// Workspace represents a collection of git repositories sharing a root directory.
type Workspace struct {
	Root  string
	Repos []Repo
}

// RepoByAlias returns the repo with the exact alias, or nil.
func (w *Workspace) RepoByAlias(alias string) *Repo {
	for i := range w.Repos {
		if w.Repos[i].Alias == alias {
			return &w.Repos[i]
		}
	}
	return nil
}

// ResolveAlias finds a repo by exact, case-insensitive, or substring match.
func (w *Workspace) ResolveAlias(query string) *Repo {
	lower := strings.ToLower(query)

	// Exact match
	for i := range w.Repos {
		if w.Repos[i].Alias == query {
			return &w.Repos[i]
		}
	}

	// Case-insensitive exact match
	for i := range w.Repos {
		if strings.ToLower(w.Repos[i].Alias) == lower {
			return &w.Repos[i]
		}
	}

	// Substring match
	for i := range w.Repos {
		if strings.Contains(strings.ToLower(w.Repos[i].Alias), lower) {
			return &w.Repos[i]
		}
	}

	return nil
}

// Aliases returns all repo aliases as a slice.
func (w *Workspace) Aliases() []string {
	aliases := make([]string, len(w.Repos))
	for i, r := range w.Repos {
		aliases[i] = r.Alias
	}
	return aliases
}
