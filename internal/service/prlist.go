package service

import (
	"sort"

	"github.com/tSquaredd/work-cli/internal/github"
)

// StandalonePR represents an open PR not associated with a detected worktree.
type StandalonePR struct {
	RepoAlias    string
	RepoDir      string
	Number       int
	Title        string
	Author       string
	URL          string
	HeadBranch   string
	HeadSHA      string
	ReviewStatus string
	CommentCount int
	Additions    int
	Deletions    int
	IsMine       bool
}

// StandalonePRs collects open PRs across all workspace repos that aren't associated
// with any worktree. Returns them split into "mine" (user's PRs) and "others".
func (s *WorkService) StandalonePRs(tasks []TaskView) (mine []StandalonePR, others []StandalonePR, err error) {
	currentUser, err := github.GetCurrentUser()
	if err != nil {
		return nil, nil, err
	}

	// Build set of worktree branches per repo to filter out
	type repoKey struct {
		alias  string
		branch string
	}
	wtBranches := make(map[repoKey]bool)
	for _, t := range tasks {
		for _, wt := range t.Worktrees {
			wtBranches[repoKey{alias: wt.Alias, branch: wt.Branch}] = true
		}
	}

	for _, repo := range s.Workspace.Repos {
		prs, listErr := github.ListOpenPRs(repo.Path)
		if listErr != nil {
			continue // skip repos where listing fails
		}

		for _, pr := range prs {
			// Filter out PRs whose branch matches a worktree
			if wtBranches[repoKey{alias: repo.Alias, branch: pr.HeadBranch}] {
				continue
			}

			sp := StandalonePR{
				RepoAlias:    repo.Alias,
				RepoDir:      repo.Path,
				Number:       pr.Number,
				Title:        pr.Title,
				Author:       pr.Author,
				URL:          pr.URL,
				HeadBranch:   pr.HeadBranch,
				HeadSHA:      pr.HeadSHA,
				ReviewStatus: pr.ReviewStatus,
				CommentCount: pr.CommentCount,
				Additions:    pr.Additions,
				Deletions:    pr.Deletions,
				IsMine:       pr.Author == currentUser,
			}

			if sp.IsMine {
				mine = append(mine, sp)
			} else {
				others = append(others, sp)
			}
		}
	}

	// Sort each by most recently updated (PRSummary.UpdatedAt is in the source;
	// we sort by number descending as a reasonable proxy since we don't carry UpdatedAt).
	sort.Slice(mine, func(i, j int) bool { return mine[i].Number > mine[j].Number })
	sort.Slice(others, func(i, j int) bool { return others[i].Number > others[j].Number })

	return mine, others, nil
}
