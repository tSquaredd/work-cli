package service

import (
	"github.com/tSquaredd/work-cli/internal/prstate"
	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/workspace"
	"github.com/tSquaredd/work-cli/internal/worktree"
)

// WorkService aggregates task data with session status.
// It serves as the query layer for both the TUI dashboard and future web UI.
type WorkService struct {
	Workspace   *workspace.Workspace
	Tracker     *session.Tracker
	PRStore     *prstate.Store
	GHAvailable bool
}

// New creates a WorkService for the given workspace.
func New(ws *workspace.Workspace, tracker *session.Tracker) *WorkService {
	return &WorkService{
		Workspace: ws,
		Tracker:   tracker,
	}
}

// Tasks returns all tasks with enriched view data including session status.
func (s *WorkService) Tasks() []TaskView {
	tasks := worktree.CollectTasks(s.Workspace)
	views := make([]TaskView, len(tasks))

	for i, t := range tasks {
		tv := TaskView{
			Name:      t.Name,
			Worktrees: make([]WorktreeView, len(t.Worktrees)),
		}

		for j, wt := range t.Worktrees {
			wv := WorktreeView{
				Alias:    wt.Alias,
				Branch:   wt.Branch,
				Status:   wt.Status,
				Dir:      wt.Dir,
				DiffStat: DiffStat(wt.Dir),
			}
			tv.Worktrees[j] = wv
		}

		// Check session status
		if s.Tracker != nil {
			if rec, ok := s.Tracker.Get(t.Name); ok {
				tv.HasSession = true
				tv.SessionPID = rec.PID
				tv.SessionLaunchedAt = rec.LaunchedAt
			}
		}

		// Enrich with cached PR data (fast, no API calls)
		if s.PRStore != nil {
			records := s.PRStore.ForTask(t.Name)
			for _, rec := range records {
				for j := range tv.Worktrees {
					if tv.Worktrees[j].Alias == rec.RepoAlias {
						tv.Worktrees[j].PR = &PRView{
							Number: rec.Number,
							URL:    rec.URL,
						}
						tv.HasPRs = true
					}
				}
			}
		}

		views[i] = tv
	}

	return views
}

// TaskDetail returns full detail for a single task, or nil if not found.
func (s *WorkService) TaskDetail(name string) *TaskView {
	for _, tv := range s.Tasks() {
		if tv.Name == name {
			return &tv
		}
	}
	return nil
}
