package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetPRDiff fetches the unified diff for a PR.
func GetPRDiff(repoDir string, prNumber int) (string, error) {
	cmd := exec.Command("gh", "-C", repoDir, "pr", "diff", fmt.Sprintf("%d", prNumber))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh pr diff: %w", err)
	}
	return string(out), nil
}

// GetPRHeadSHA fetches the HEAD commit SHA for a PR.
func GetPRHeadSHA(repoDir string, prNumber int) (string, error) {
	cmd := exec.Command("gh", "-C", repoDir, "pr", "view",
		fmt.Sprintf("%d", prNumber),
		"--json", "headRefOid",
		"--jq", ".headRefOid",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh pr view: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CreateReviewComment posts a single-line review comment on a PR.
func CreateReviewComment(repoDir string, prNumber int, commitID, filePath string, line int, side string, body string) error {
	owner, repo, err := RepoFromRemote(repoDir)
	if err != nil {
		return fmt.Errorf("resolving repo: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/comments", owner, repo, prNumber)
	cmd := exec.Command("gh", "api", "-X", "POST", endpoint,
		"-f", fmt.Sprintf("body=%s", body),
		"-f", fmt.Sprintf("path=%s", filePath),
		"-F", fmt.Sprintf("line=%d", line),
		"-f", fmt.Sprintf("side=%s", side),
		"-f", fmt.Sprintf("commit_id=%s", commitID),
	)
	cmd.Dir = repoDir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("posting review comment: %s: %w", string(out), err)
	}

	return nil
}

// CreateMultiLineReviewComment posts a multi-line review comment on a PR.
func CreateMultiLineReviewComment(repoDir string, prNumber int, commitID, filePath string, startLine, endLine int, side string, body string) error {
	owner, repo, err := RepoFromRemote(repoDir)
	if err != nil {
		return fmt.Errorf("resolving repo: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/comments", owner, repo, prNumber)
	cmd := exec.Command("gh", "api", "-X", "POST", endpoint,
		"-f", fmt.Sprintf("body=%s", body),
		"-f", fmt.Sprintf("path=%s", filePath),
		"-F", fmt.Sprintf("line=%d", endLine),
		"-f", fmt.Sprintf("side=%s", side),
		"-F", fmt.Sprintf("start_line=%d", startLine),
		"-f", fmt.Sprintf("start_side=%s", side),
		"-f", fmt.Sprintf("commit_id=%s", commitID),
	)
	cmd.Dir = repoDir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("posting multi-line review comment: %s: %w", string(out), err)
	}

	return nil
}
