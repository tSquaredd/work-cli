package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// PRInfo holds structured data about a GitHub pull request.
type PRInfo struct {
	Number       int
	Title        string
	URL          string
	State        string // "OPEN", "MERGED", "CLOSED"
	ReviewStatus string // "APPROVED", "CHANGES_REQUESTED", "REVIEW_REQUIRED", ""
	CommentCount int
	HeadBranch   string
	BaseBranch   string
	UpdatedAt    time.Time
}

// ghPRJSON maps the JSON output of `gh pr view --json ...`.
type ghPRJSON struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	ReviewDecision string    `json:"reviewDecision"`
	HeadRefName    string    `json:"headRefName"`
	BaseRefName    string    `json:"baseRefName"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Comments       []struct {
		ID string `json:"id"`
	} `json:"comments"`
}

func prInfoFromJSON(j ghPRJSON) PRInfo {
	return PRInfo{
		Number:       j.Number,
		Title:        j.Title,
		URL:          j.URL,
		State:        j.State,
		ReviewStatus: j.ReviewDecision,
		CommentCount: len(j.Comments),
		HeadBranch:   j.HeadRefName,
		BaseBranch:   j.BaseRefName,
		UpdatedAt:    j.UpdatedAt,
	}
}

// CreateInBrowser opens the browser to create a new PR for the current branch.
func CreateInBrowser(dir string) error {
	cmd := exec.Command("gh", "-C", dir, "pr", "create", "--web")
	return cmd.Run()
}

// FindPRForBranch finds the PR associated with the current branch in the given directory.
func FindPRForBranch(dir string) (*PRInfo, error) {
	cmd := exec.Command("gh", "-C", dir, "pr", "view",
		"--json", "number,title,url,state,reviewDecision,comments,headRefName,baseRefName,updatedAt",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err // no PR found or gh error
	}

	var j ghPRJSON
	if err := json.Unmarshal(out, &j); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	info := prInfoFromJSON(j)
	return &info, nil
}

// ghPRListJSON maps items from `gh pr list --json ...`.
type ghPRListJSON struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	ReviewDecision string    `json:"reviewDecision"`
	HeadRefName    string    `json:"headRefName"`
	BaseRefName    string    `json:"baseRefName"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Comments       []struct {
		ID string `json:"id"`
	} `json:"comments"`
}

// ListPRsForBranches returns PRs matching the given branch names.
func ListPRsForBranches(dir string, branches []string) ([]PRInfo, error) {
	cmd := exec.Command("gh", "-C", dir, "pr", "list",
		"--state", "all",
		"--json", "number,title,url,state,reviewDecision,comments,headRefName,baseRefName,updatedAt",
		"--limit", "100",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var items []ghPRListJSON
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	// Build lookup set
	branchSet := make(map[string]bool, len(branches))
	for _, b := range branches {
		branchSet[b] = true
	}

	var result []PRInfo
	for _, item := range items {
		if branchSet[item.HeadRefName] {
			result = append(result, PRInfo{
				Number:       item.Number,
				Title:        item.Title,
				URL:          item.URL,
				State:        item.State,
				ReviewStatus: item.ReviewDecision,
				CommentCount: len(item.Comments),
				HeadBranch:   item.HeadRefName,
				BaseBranch:   item.BaseRefName,
				UpdatedAt:    item.UpdatedAt,
			})
		}
	}

	return result, nil
}

// OpenInBrowser opens the PR for the current branch in the default browser.
func OpenInBrowser(dir string) error {
	cmd := exec.Command("gh", "-C", dir, "pr", "view", "--web")
	return cmd.Run()
}
