package service

import (
	"fmt"
	"sort"
	"strings"

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

// StandalonePRs collects all open PRs across workspace repos.
// Returns them split into "mine" (user's PRs) and "others".
func (s *WorkService) StandalonePRs(tasks []TaskView) (mine []StandalonePR, others []StandalonePR, err error) {
	currentUser, err := github.GetCurrentUser()
	if err != nil {
		return nil, nil, err
	}

	var errs []string
	for _, repo := range s.Workspace.Repos {
		prs, listErr := github.ListOpenPRs(repo.Path)
		if listErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", repo.Alias, listErr))
			continue
		}

		for _, pr := range prs {
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

	// If all repos failed and we got no PRs, surface the errors
	if len(mine) == 0 && len(others) == 0 && len(errs) > 0 {
		return nil, nil, fmt.Errorf("PR list failed: %s", strings.Join(errs, "; "))
	}

	// Sort by number descending as a reasonable proxy for most recently updated.
	sort.Slice(mine, func(i, j int) bool { return mine[i].Number > mine[j].Number })
	sort.Slice(others, func(i, j int) bool { return others[i].Number > others[j].Number })

	return mine, others, nil
}
