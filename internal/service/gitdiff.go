package service

import (
	"os/exec"
	"strings"
)

// Diff returns the most relevant diff for a worktree directory.
// It cascades through:
//  1. Uncommitted changes (staged + unstaged) via `git diff HEAD`
//  2. Unpushed commits vs remote branch via `git diff origin/<branch>..HEAD`
//  3. Empty string if nothing differs
func Diff(dir string) string {
	// First: uncommitted changes (staged + unstaged)
	if d := gitDiff(dir, "HEAD"); d != "" {
		return d
	}

	// Second: unpushed commits compared to remote
	if d := diffVsRemote(dir); d != "" {
		return d
	}

	return ""
}

// DiffStat returns a short stat summary using the same cascade as Diff.
func DiffStat(dir string) string {
	// Uncommitted changes
	if s := gitDiffStat(dir, "HEAD"); s != "" {
		return s
	}

	// Unpushed commits
	branch := currentBranch(dir)
	if branch != "" {
		if s := gitDiffStat(dir, "origin/"+branch+"..HEAD"); s != "" {
			return s
		}
	}

	return ""
}

// DiffFiles returns the list of changed files using the same cascade.
func DiffFiles(dir string) []string {
	// Uncommitted
	if f := gitDiffNameOnly(dir, "HEAD"); len(f) > 0 {
		return f
	}

	// Unpushed
	branch := currentBranch(dir)
	if branch != "" {
		if f := gitDiffNameOnly(dir, "origin/"+branch+"..HEAD"); len(f) > 0 {
			return f
		}
	}

	return nil
}

// gitDiff runs git diff against a ref and returns the output.
func gitDiff(dir, ref string) string {
	cmd := exec.Command("git", "-C", dir, "diff", ref)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// diffVsRemote returns the diff of unpushed commits against the remote branch.
func diffVsRemote(dir string) string {
	branch := currentBranch(dir)
	if branch == "" {
		return ""
	}

	// Check that the remote branch exists
	check := exec.Command("git", "-C", dir, "rev-parse", "--verify", "origin/"+branch)
	if check.Run() != nil {
		// No remote branch — diff all commits on this branch
		// Find the merge base with the default branch
		for _, base := range []string{"origin/main", "origin/master", "origin/develop"} {
			mbCmd := exec.Command("git", "-C", dir, "merge-base", base, "HEAD")
			if mb, err := mbCmd.Output(); err == nil {
				return gitDiff(dir, strings.TrimSpace(string(mb)))
			}
		}
		return ""
	}

	return gitDiff(dir, "origin/"+branch+"..HEAD")
}

// gitDiffStat runs git diff --stat and returns the summary line.
func gitDiffStat(dir, ref string) string {
	cmd := exec.Command("git", "-C", dir, "diff", "--stat", ref)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// gitDiffNameOnly runs git diff --name-only and returns the file list.
func gitDiffNameOnly(dir, ref string) []string {
	cmd := exec.Command("git", "-C", dir, "diff", "--name-only", ref)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// currentBranch returns the branch name or empty string for detached HEAD.
func currentBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
