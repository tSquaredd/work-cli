package worktree

import (
	"os"
	"path/filepath"

	"github.com/tSquaredd/work-cli/internal/workspace"
)

// RepoWorktree represents a single repo's worktree within a task.
type RepoWorktree struct {
	Alias  string
	Branch string
	Status Status
	Dir    string
}

// Task represents a named group of worktrees (one per repo).
type Task struct {
	Name      string
	Worktrees []RepoWorktree
}

// CollectTasks scans the workspace for worktrees grouped by task name.
// Checks both new-style (<workspace>/.worktrees/<task>/<repo>/) and
// old-style (<repo>/.claude/worktrees/<task>/) locations.
func CollectTasks(ws *workspace.Workspace) []Task {
	taskMap := make(map[string]*Task)
	var taskOrder []string

	addWorktree := func(taskName string, wt RepoWorktree) {
		t, exists := taskMap[taskName]
		if !exists {
			t = &Task{Name: taskName}
			taskMap[taskName] = t
			taskOrder = append(taskOrder, taskName)
		}
		t.Worktrees = append(t.Worktrees, wt)
	}

	// New-style: <workspace>/.worktrees/<task>/<repo>/
	wtBase := filepath.Join(ws.Root, ".worktrees")
	if entries, err := os.ReadDir(wtBase); err == nil {
		for _, taskEntry := range entries {
			if !taskEntry.IsDir() {
				continue
			}
			taskName := taskEntry.Name()
			taskDir := filepath.Join(wtBase, taskName)

			repoEntries, err := os.ReadDir(taskDir)
			if err != nil {
				continue
			}
			for _, repoEntry := range repoEntries {
				if !repoEntry.IsDir() {
					continue
				}
				repoName := repoEntry.Name()
				// Verify this is a known repo
				if ws.RepoByAlias(repoName) == nil {
					continue
				}

				repoWtDir := filepath.Join(taskDir, repoName)
				info := Inspect(repoWtDir)
				addWorktree(taskName, RepoWorktree{
					Alias:  repoName,
					Branch: info.Branch,
					Status: info.Status,
					Dir:    repoWtDir,
				})
			}
		}
	}

	// Old-style: <repo>/.claude/worktrees/<task>/
	for _, repo := range ws.Repos {
		claudeWtDir := filepath.Join(repo.Path, ".claude", "worktrees")
		taskEntries, err := os.ReadDir(claudeWtDir)
		if err != nil {
			continue
		}
		for _, taskEntry := range taskEntries {
			if !taskEntry.IsDir() {
				continue
			}
			taskName := taskEntry.Name()
			wtDir := filepath.Join(claudeWtDir, taskName)
			info := Inspect(wtDir)
			addWorktree(taskName, RepoWorktree{
				Alias:  repo.Alias,
				Branch: info.Branch,
				Status: info.Status,
				Dir:    wtDir,
			})
		}
	}

	// Build ordered result
	tasks := make([]Task, 0, len(taskOrder))
	for _, name := range taskOrder {
		tasks = append(tasks, *taskMap[name])
	}
	return tasks
}

// FindTaskDirs returns the worktree directories for a named task.
func FindTaskDirs(ws *workspace.Workspace, taskName string) []string {
	tasks := CollectTasks(ws)
	for _, t := range tasks {
		if t.Name == taskName {
			dirs := make([]string, len(t.Worktrees))
			for i, wt := range t.Worktrees {
				dirs[i] = wt.Dir
			}
			return dirs
		}
	}
	return nil
}
