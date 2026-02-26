package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Discover finds the workspace root and all git repositories within it.
func Discover() (*Workspace, error) {
	root, err := findWorkspaceRoot()
	if err != nil {
		return nil, err
	}
	return discoverRepos(root)
}

// DiscoverFrom finds the workspace root starting from the given directory.
func DiscoverFrom(dir string) (*Workspace, error) {
	root, err := findWorkspaceRootFrom(dir)
	if err != nil {
		return nil, err
	}
	return discoverRepos(root)
}

func discoverRepos(root string) (*Workspace, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading workspace root: %w", err)
	}

	ws := &Workspace{Root: root}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		child := filepath.Join(root, e.Name())
		gitDir := filepath.Join(child, ".git")
		if _, err := os.Stat(gitDir); err != nil {
			continue
		}
		repo := Repo{
			Alias:       e.Name(),
			Path:        child,
			Prefix:      AutoPrefix(e.Name()),
			Description: AutoDescription(child),
		}
		ws.Repos = append(ws.Repos, repo)
	}

	if len(ws.Repos) == 0 {
		return nil, fmt.Errorf("no git repositories found in %s", root)
	}

	sort.Slice(ws.Repos, func(i, j int) bool {
		return ws.Repos[i].Alias < ws.Repos[j].Alias
	})

	return ws, nil
}

func findWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return findWorkspaceRootFrom(cwd)
}

func findWorkspaceRootFrom(dir string) (string, error) {
	// Strategy 1: dir has child directories with .git
	if count := countChildRepos(dir); count >= 1 {
		return dir, nil
	}

	// Strategy 2: We're inside a git repo — go up to find siblings
	gitRoot := gitToplevel(dir)
	if gitRoot != "" {
		parent := filepath.Dir(gitRoot)
		if count := countChildRepos(parent); count >= 1 {
			return parent, nil
		}
		// Single repo — parent of git root is still the workspace
		return parent, nil
	}

	return "", fmt.Errorf("no git repositories found — run from a directory containing git repos, or from inside a git repo")
}

func countChildRepos(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitDir := filepath.Join(dir, e.Name(), ".git")
		if _, err := os.Stat(gitDir); err == nil {
			count++
		}
	}
	return count
}

func gitToplevel(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
