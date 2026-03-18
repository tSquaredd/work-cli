package worktree

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Status represents the git status of a worktree.
type Status int

const (
	StatusClean    Status = iota
	StatusDirty           // Uncommitted changes or untracked files
	StatusUnpushed        // Commits not on remote
	StatusPushed          // Branch exists on remote, up to date
)

func (s Status) String() string {
	switch s {
	case StatusDirty:
		return "DIRTY"
	case StatusPushed:
		return "PUSHED"
	case StatusUnpushed:
		return "UNPUSHED"
	default:
		return "CLEAN"
	}
}

// GitInfo holds branch and status for a worktree directory.
type GitInfo struct {
	Branch string
	Status Status
}

// Inspect returns the git branch and status for a worktree directory.
func Inspect(dir string) GitInfo {
	branch := Branch(dir)
	status := InspectStatus(dir)
	return GitInfo{Branch: branch, Status: status}
}

// Branch returns the current branch name for a directory, or "detached".
func Branch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "detached"
	}
	b := strings.TrimSpace(string(out))
	if b == "" {
		return "detached"
	}
	return b
}

// InspectStatus determines the worktree status: PUSHED > UNPUSHED > DIRTY > CLEAN.
func InspectStatus(dir string) Status {
	pushed := IsPushed(dir)
	unpushed := HasUnpushed(dir)
	if pushed && !unpushed {
		return StatusPushed
	}
	if unpushed {
		return StatusUnpushed
	}
	if IsDirty(dir) {
		return StatusDirty
	}
	return StatusClean
}

// IsDirty returns true if there are uncommitted changes or untracked files.
func IsDirty(dir string) bool {
	return hasDirtyChanges(dir) || hasUntracked(dir)
}

func hasDirtyChanges(dir string) bool {
	// Check staged changes
	cmd1 := exec.Command("git", "-C", dir, "diff", "--quiet")
	err1 := cmd1.Run()

	// Check unstaged changes
	cmd2 := exec.Command("git", "-C", dir, "diff", "--cached", "--quiet")
	err2 := cmd2.Run()

	return err1 != nil || err2 != nil
}

func hasUntracked(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "ls-files", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// IsPushed returns true if the branch exists on the remote.
func IsPushed(dir string) bool {
	branch := Branch(dir)
	if branch == "detached" {
		return false
	}
	cmd := exec.Command("git", "-C", dir, "rev-parse", "origin/"+branch)
	return cmd.Run() == nil
}

// HasUnpushed returns true if there are local commits not pushed to remote.
func HasUnpushed(dir string) bool {
	branch := Branch(dir)
	if branch == "detached" {
		return false
	}

	// Check if remote branch exists
	cmd := exec.Command("git", "-C", dir, "rev-parse", "origin/"+branch)
	if cmd.Run() != nil {
		// No remote branch — check if we have any commits
		countCmd := exec.Command("git", "-C", dir, "rev-list", "--count", "HEAD")
		out, err := countCmd.Output()
		if err != nil {
			return false
		}
		count, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		return count > 0
	}

	// Remote exists — check ahead count
	countCmd := exec.Command("git", "-C", dir, "rev-list", "--count", "origin/"+branch+".."+branch)
	out, err := countCmd.Output()
	if err != nil {
		return false
	}
	ahead, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return ahead > 0
}

// RecentBranches returns remote branch names sorted by most recent commit.
func RecentBranches(dir string, limit int) []string {
	cmd := exec.Command("git", "-C", dir, "branch", "-r",
		"--sort=-committerdate",
		"--format=%(refname:short)")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip origin/ prefix
		line = strings.TrimPrefix(line, "origin/")
		if line == "HEAD" {
			continue
		}
		branches = append(branches, line)
		if len(branches) >= limit {
			break
		}
	}
	return branches
}

// HasRemoteBranch checks if origin/{branch} exists locally (no network call).
func HasRemoteBranch(repoDir, branch string) bool {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", "origin/"+branch)
	return cmd.Run() == nil
}

// Checkout switches to the given branch.
func Checkout(dir, branch string) error {
	cmd := exec.Command("git", "-C", dir, "checkout", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkout %s: %s", branch, strings.TrimSpace(string(out)))
	}
	return nil
}

// Pull pulls the current branch from origin.
func Pull(dir string) error {
	cmd := exec.Command("git", "-C", dir, "pull")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pull: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// FetchAll runs git fetch to update all remote-tracking refs. Best-effort; errors are ignored.
func FetchAll(dir string) {
	_ = exec.Command("git", "-C", dir, "fetch").Run()
}

// AllBranches returns all branch names (local + remote origin/*) for the repo at dir,
// deduplicated and sorted alphabetically. Call FetchAll before this to include
// branches that only exist on the remote.
func AllBranches(dir string) []string {
	seen := make(map[string]bool)

	// Local branches
	if out, err := exec.Command("git", "-C", dir, "branch",
		"--format=%(refname:short)").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				seen[line] = true
			}
		}
	}

	// Remote-tracking branches (origin/*)
	if out, err := exec.Command("git", "-C", dir, "branch", "-r",
		"--format=%(refname:short)").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimPrefix(strings.TrimSpace(line), "origin/")
			if line != "" && line != "HEAD" {
				seen[line] = true
			}
		}
	}

	branches := make([]string, 0, len(seen))
	for b := range seen {
		branches = append(branches, b)
	}
	sort.Strings(branches)
	return branches
}

// LocalBranches returns local branch names sorted by most recent commit.
func LocalBranches(dir string) []string {
	cmd := exec.Command("git", "-C", dir, "branch",
		"--format=%(refname:short)", "--sort=-committerdate")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches
}

// WorktreeBranches returns a set of branch names currently checked out in any
// worktree of the repo at dir (including the main worktree).
func WorktreeBranches(dir string) map[string]bool {
	cmd := exec.Command("git", "-C", dir, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	result := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "branch ") {
			b := strings.TrimPrefix(line, "branch ")
			b = strings.TrimPrefix(b, "refs/heads/")
			result[b] = true
		}
	}
	return result
}

// CurrentBranch returns the current branch of a repo (not a worktree).
func CurrentBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	b := strings.TrimSpace(string(out))
	if b == "" {
		return "main"
	}
	return b
}
