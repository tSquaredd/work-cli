package service

import (
	"time"

	"github.com/tSquaredd/work-cli/internal/worktree"
)

// TaskView is an enriched view of a task with session and diff info.
type TaskView struct {
	Name      string
	Worktrees []WorktreeView

	// Session info
	HasSession        bool
	SessionPID        int
	SessionLaunchedAt time.Time
}

// WorktreeView is an enriched view of a single worktree with diff stats.
type WorktreeView struct {
	Alias    string
	Branch   string
	Status   worktree.Status
	Dir      string
	DiffStat string // e.g. "3 files changed, 12 insertions(+), 4 deletions(-)"
}

// Dirs returns all worktree directories for this task.
func (tv *TaskView) Dirs() []string {
	dirs := make([]string, len(tv.Worktrees))
	for i, wt := range tv.Worktrees {
		dirs[i] = wt.Dir
	}
	return dirs
}

// OverallStatus returns the highest-priority status across all worktrees.
func (tv *TaskView) OverallStatus() worktree.Status {
	best := worktree.StatusClean
	for _, wt := range tv.Worktrees {
		if wt.Status > best {
			best = wt.Status
		}
	}
	return best
}
