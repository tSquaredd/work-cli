package service

import (
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/prstate"
)

// PREnricher populates PR data on task views using cached state and live gh lookups.
type PREnricher struct {
	Store       *prstate.Store
	ghAvailable bool
}

// NewPREnricher creates a PREnricher. If gh is not available, live lookups are disabled.
func NewPREnricher(store *prstate.Store, ghAvailable bool) *PREnricher {
	return &PREnricher{
		Store:       store,
		ghAvailable: ghAvailable,
	}
}

// EnrichTasks populates PRView fields from cached prstate records (no API calls).
func (e *PREnricher) EnrichTasks(tasks []TaskView) []TaskView {
	if e.Store == nil {
		return tasks
	}

	for i := range tasks {
		records := e.Store.ForTask(tasks[i].Name)
		for _, rec := range records {
			for j := range tasks[i].Worktrees {
				wt := &tasks[i].Worktrees[j]
				if wt.Alias == rec.RepoAlias {
					newComments := 0
					if wt.PR != nil && wt.PR.CommentCount > rec.LastComments {
						newComments = wt.PR.CommentCount - rec.LastComments
					}
					if wt.PR == nil {
						wt.PR = &PRView{
							Number: rec.Number,
							URL:    rec.URL,
						}
					}
					wt.PR.NewComments = newComments
					tasks[i].HasPRs = true
				}
			}
		}
	}

	return tasks
}

// RefreshPRStatus calls gh pr view for each known open PR, updates prstate, and enriches tasks.
func (e *PREnricher) RefreshPRStatus(tasks []TaskView) []TaskView {
	if e.Store == nil || !e.ghAvailable {
		return tasks
	}

	for i := range tasks {
		for j := range tasks[i].Worktrees {
			wt := &tasks[i].Worktrees[j]
			if wt.PR == nil || wt.PR.Number == 0 {
				continue
			}

			info, err := github.FindPRForBranch(wt.Dir)
			if err != nil {
				continue // silently skip, retain cached data
			}

			// Get cached record for new comment computation
			rec, hasRec := e.Store.Get(tasks[i].Name, wt.Alias)
			newComments := 0
			if hasRec && info.CommentCount > rec.LastComments {
				newComments = info.CommentCount - rec.LastComments
			}

			wt.PR = &PRView{
				Number:       info.Number,
				Title:        info.Title,
				URL:          info.URL,
				State:        info.State,
				ReviewStatus: info.ReviewStatus,
				CommentCount: info.CommentCount,
				NewComments:  newComments,
			}
			tasks[i].HasPRs = true

			// Update stored record with latest data (but preserve LastViewed/LastComments)
			_ = e.Store.Save(prstate.PRRecord{
				TaskName:     tasks[i].Name,
				RepoAlias:    wt.Alias,
				Number:       info.Number,
				URL:          info.URL,
				State:        info.State,
				ReviewStatus: info.ReviewStatus,
				Title:        info.Title,
				LastViewed:   rec.LastViewed,
				LastComments: rec.LastComments,
			})
		}
	}

	return tasks
}

// DiscoverPRs checks worktrees without known PRs to see if a PR exists.
func (e *PREnricher) DiscoverPRs(tasks []TaskView) []TaskView {
	if e.Store == nil || !e.ghAvailable {
		return tasks
	}

	for i := range tasks {
		for j := range tasks[i].Worktrees {
			wt := &tasks[i].Worktrees[j]
			if wt.PR != nil {
				continue // already has a known PR
			}

			info, err := github.FindPRForBranch(wt.Dir)
			if err != nil || info == nil {
				continue // no PR exists
			}

			// Get cached record for new comment computation
			rec, hasRec := e.Store.Get(tasks[i].Name, wt.Alias)
			newComments := 0
			if hasRec && info.CommentCount > rec.LastComments {
				newComments = info.CommentCount - rec.LastComments
			}

			wt.PR = &PRView{
				Number:       info.Number,
				Title:        info.Title,
				URL:          info.URL,
				State:        info.State,
				ReviewStatus: info.ReviewStatus,
				CommentCount: info.CommentCount,
				NewComments:  newComments,
			}
			tasks[i].HasPRs = true

			// Save discovered PR to state
			_ = e.Store.Save(prstate.PRRecord{
				TaskName:     tasks[i].Name,
				RepoAlias:    wt.Alias,
				Number:       info.Number,
				URL:          info.URL,
				State:        info.State,
				ReviewStatus: info.ReviewStatus,
				Title:        info.Title,
				LastViewed:   rec.LastViewed,
				LastComments: rec.LastComments,
			})
		}
	}

	return tasks
}
