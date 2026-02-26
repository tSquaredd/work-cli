package service

import (
	"os/exec"
	"strings"
)

// Diff returns the full git diff output for a worktree directory.
func Diff(dir string) string {
	cmd := exec.Command("git", "-C", dir, "diff")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// DiffStat returns a short stat summary (e.g. "3 files changed, 12 insertions(+)").
func DiffStat(dir string) string {
	cmd := exec.Command("git", "-C", dir, "diff", "--stat")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return ""
	}
	// The last line of --stat is the summary
	return strings.TrimSpace(lines[len(lines)-1])
}

// DiffCached returns the staged diff for a worktree directory.
func DiffCached(dir string) string {
	cmd := exec.Command("git", "-C", dir, "diff", "--cached")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// DiffFiles returns the list of changed files in a worktree.
func DiffFiles(dir string) []string {
	cmd := exec.Command("git", "-C", dir, "diff", "--name-only")
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
