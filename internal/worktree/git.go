package worktree

import (
	"os/exec"
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
