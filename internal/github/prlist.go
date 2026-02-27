package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PRSummary holds basic info about an open PR, used for listing all PRs in a repo.
type PRSummary struct {
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
	UpdatedAt    time.Time
}

// ghPRListFullJSON maps the JSON output of `gh pr list --json ...` for PR summaries.
type ghPRListFullJSON struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	ReviewDecision string    `json:"reviewDecision"`
	HeadRefName    string    `json:"headRefName"`
	HeadRefOid     string    `json:"headRefOid"`
	Additions      int       `json:"additions"`
	Deletions      int       `json:"deletions"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Author         struct {
		Login string `json:"login"`
	} `json:"author"`
	Comments []struct {
		ID string `json:"id"`
	} `json:"comments"`
}

// ListOpenPRs lists all open PRs for the repo at repoDir.
func ListOpenPRs(repoDir string) ([]PRSummary, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--state", "open",
		"--json", "number,title,author,url,headRefName,headRefOid,reviewDecision,comments,additions,deletions,updatedAt",
		"--limit", "100",
	)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Take only the first line of output to avoid flooding with gh's usage text
		errLine := strings.TrimSpace(string(out))
		if i := strings.Index(errLine, "\n"); i >= 0 {
			errLine = strings.TrimSpace(errLine[:i])
		}
		return nil, fmt.Errorf("gh pr list: %s: %w", errLine, err)
	}

	var items []ghPRListFullJSON
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("parsing gh pr list output: %w", err)
	}

	result := make([]PRSummary, len(items))
	for i, item := range items {
		result[i] = PRSummary{
			Number:       item.Number,
			Title:        item.Title,
			Author:       item.Author.Login,
			URL:          item.URL,
			HeadBranch:   item.HeadRefName,
			HeadSHA:      item.HeadRefOid,
			ReviewStatus: item.ReviewDecision,
			CommentCount: len(item.Comments),
			Additions:    item.Additions,
			Deletions:    item.Deletions,
			UpdatedAt:    item.UpdatedAt,
		}
	}

	return result, nil
}

// GetCurrentUser returns the authenticated GitHub username.
func GetCurrentUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh api user: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
