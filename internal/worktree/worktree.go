package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tSquaredd/work-cli/internal/workspace"
)

// CreateConfig holds parameters for creating a worktree.
type CreateConfig struct {
	RepoDir    string // Path to the main repo
	WorktreeDir string // Destination directory for the worktree
	Branch     string // New branch name
	BaseBranch string // Branch to fork from (e.g., "origin/develop")
}

// CreateResult holds the outcome of a worktree creation.
type CreateResult struct {
	Dir      string
	Branch   string
	Created  bool   // false if already existed
	Attached bool   // true if attached to existing branch
	Error    error
}

// Create creates a new worktree. Tries creating a new branch first, then
// attaching to an existing branch as fallback.
func Create(cfg CreateConfig) CreateResult {
	// Already exists
	if _, err := os.Stat(cfg.WorktreeDir); err == nil {
		return CreateResult{Dir: cfg.WorktreeDir, Branch: cfg.Branch, Created: false}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.WorktreeDir), 0o755); err != nil {
		return CreateResult{Error: fmt.Errorf("creating parent directory: %w", err)}
	}

	// Try creating a new branch from the base
	cmd := exec.Command("git", "-C", cfg.RepoDir, "worktree", "add",
		"-b", cfg.Branch, cfg.WorktreeDir, "origin/"+cfg.BaseBranch)
	if err := cmd.Run(); err == nil {
		return CreateResult{Dir: cfg.WorktreeDir, Branch: cfg.Branch, Created: true}
	}

	// Fallback: attach to existing branch
	cmd = exec.Command("git", "-C", cfg.RepoDir, "worktree", "add",
		cfg.WorktreeDir, cfg.Branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return CreateResult{Error: fmt.Errorf("creating worktree: %s", strings.TrimSpace(string(out)))}
	}

	return CreateResult{Dir: cfg.WorktreeDir, Branch: cfg.Branch, Created: true, Attached: true}
}

// CreateDirect creates a worktree directly from HEAD (for direct launch).
func CreateDirect(repoDir, wtDir, branch string) CreateResult {
	if _, err := os.Stat(wtDir); err == nil {
		return CreateResult{Dir: wtDir, Branch: branch, Created: false}
	}

	if err := os.MkdirAll(filepath.Dir(wtDir), 0o755); err != nil {
		return CreateResult{Error: fmt.Errorf("creating parent directory: %w", err)}
	}

	// Try new branch from HEAD
	cmd := exec.Command("git", "-C", repoDir, "worktree", "add",
		"-b", branch, wtDir, "HEAD")
	if err := cmd.Run(); err == nil {
		return CreateResult{Dir: wtDir, Branch: branch, Created: true}
	}

	// Fallback: existing branch
	cmd = exec.Command("git", "-C", repoDir, "worktree", "add", wtDir, branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return CreateResult{Error: fmt.Errorf("creating worktree: %s", strings.TrimSpace(string(out)))}
	}
	return CreateResult{Dir: wtDir, Branch: branch, Created: true, Attached: true}
}

// Fetch runs git fetch for a specific branch.
func Fetch(repoDir, branch string) error {
	cmd := exec.Command("git", "-C", repoDir, "fetch", "origin", branch)
	return cmd.Run()
}

// Remove removes a worktree and optionally deletes the branch.
func Remove(repoDir, wtDir string, deleteBranch bool) error {
	branch := Branch(wtDir)

	cmd := exec.Command("git", "-C", repoDir, "worktree", "remove", "--force", wtDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}

	if deleteBranch && branch != "detached" {
		delCmd := exec.Command("git", "-C", repoDir, "branch", "-D", branch)
		_ = delCmd.Run() // Best effort
	}

	return nil
}

// CleanRemove removes a worktree using soft branch delete (only merged branches).
func CleanRemove(repoDir, wtDir string) error {
	branch := Branch(wtDir)

	cmd := exec.Command("git", "-C", repoDir, "worktree", "remove", wtDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}

	if branch != "detached" {
		delCmd := exec.Command("git", "-C", repoDir, "branch", "-d", branch)
		_ = delCmd.Run() // Best effort — -d only deletes if merged
	}

	return nil
}

// CleanupTaskDir removes the task directory and all remaining contents.
func CleanupTaskDir(wsRoot, taskName string) {
	taskDir := filepath.Join(wsRoot, ".worktrees", taskName)
	worktreesBase := filepath.Join(wsRoot, ".worktrees")
	absTask, err1 := filepath.Abs(taskDir)
	absBase, err2 := filepath.Abs(worktreesBase)
	if err1 != nil || err2 != nil || !strings.HasPrefix(absTask, absBase+string(filepath.Separator)) {
		return
	}
	_ = os.RemoveAll(taskDir)
}

// WorktreeDir returns the standard worktree path for a repo/task combination.
func WorktreeDir(ws *workspace.Workspace, taskName, repoAlias string) string {
	return filepath.Join(ws.Root, ".worktrees", taskName, repoAlias)
}

// ResolveWorktreeDir finds the worktree directory for a repo/task, checking
// both new and old locations.
func ResolveWorktreeDir(ws *workspace.Workspace, taskName, repoAlias string) string {
	// New location
	dir := WorktreeDir(ws, taskName, repoAlias)
	if _, err := os.Stat(dir); err == nil {
		return dir
	}

	// Old location
	repo := ws.RepoByAlias(repoAlias)
	if repo != nil {
		oldDir := filepath.Join(repo.Path, ".claude", "worktrees", taskName)
		if _, err := os.Stat(oldDir); err == nil {
			return oldDir
		}
	}

	return dir // Return new-style path even if it doesn't exist
}
