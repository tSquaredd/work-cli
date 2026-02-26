package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ReviewComment represents a single comment within a review thread.
type ReviewComment struct {
	ID        int
	Author    string
	Body      string
	Path      string // file path relative to repo root
	Line      int    // line number (0 if file-level)
	DiffHunk  string // diff context snippet
	CreatedAt time.Time
}

// ReviewThread represents a threaded review conversation on a specific file/line.
type ReviewThread struct {
	Path       string
	Line       int
	DiffHunk   string
	IsResolved bool
	Comments   []ReviewComment // chronological
}

// IssueComment represents a top-level (non-inline) PR comment.
type IssueComment struct {
	ID        int
	Author    string
	Body      string
	CreatedAt time.Time
}

// PRComments holds all comments for a pull request.
type PRComments struct {
	Threads       []ReviewThread
	IssueComments []IssueComment
}

// graphQL response types for unmarshaling
type graphQLResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					Nodes []graphQLThread `json:"nodes"`
				} `json:"reviewThreads"`
				Comments struct {
					Nodes []graphQLIssueComment `json:"nodes"`
				} `json:"comments"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

type graphQLThread struct {
	IsResolved bool `json:"isResolved"`
	Comments   struct {
		Nodes []graphQLComment `json:"nodes"`
	} `json:"comments"`
}

type graphQLComment struct {
	DatabaseID int    `json:"databaseId"`
	Author     struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	Path      string `json:"path"`
	Line      *int   `json:"line"`
	DiffHunk  string `json:"diffHunk"`
	CreatedAt string `json:"createdAt"`
}

type graphQLIssueComment struct {
	DatabaseID int    `json:"databaseId"`
	Author     struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

const prCommentsQuery = `query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 50) {
            nodes {
              databaseId
              author { login }
              body
              path
              line
              diffHunk
              createdAt
            }
          }
        }
      }
      comments(first: 100) {
        nodes {
          databaseId
          author { login }
          body
          createdAt
        }
      }
    }
  }
}`

// FetchPRComments fetches all review threads and issue comments for a PR.
func FetchPRComments(dir string, prNumber int) (*PRComments, error) {
	owner, repo, err := RepoFromRemote(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving repo: %w", err)
	}
	if owner == "" {
		return nil, fmt.Errorf("could not determine GitHub owner/repo from remote")
	}

	cmd := exec.Command("gh", "api", "graphql",
		"-F", fmt.Sprintf("owner=%s", owner),
		"-F", fmt.Sprintf("repo=%s", repo),
		"-F", fmt.Sprintf("number=%d", prNumber),
		"-f", fmt.Sprintf("query=%s", prCommentsQuery),
	)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api graphql: %w", err)
	}

	var resp graphQLResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing graphql response: %w", err)
	}

	result := &PRComments{}

	// Convert review threads
	for _, t := range resp.Data.Repository.PullRequest.ReviewThreads.Nodes {
		thread := ReviewThread{
			IsResolved: t.IsResolved,
		}

		for i, c := range t.Comments.Nodes {
			createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
			line := 0
			if c.Line != nil {
				line = *c.Line
			}

			comment := ReviewComment{
				ID:        c.DatabaseID,
				Author:    c.Author.Login,
				Body:      c.Body,
				Path:      c.Path,
				Line:      line,
				DiffHunk:  c.DiffHunk,
				CreatedAt: createdAt,
			}

			// Set thread-level path/line/diffhunk from first comment
			if i == 0 {
				thread.Path = c.Path
				thread.Line = line
				thread.DiffHunk = c.DiffHunk
			}

			thread.Comments = append(thread.Comments, comment)
		}

		if len(thread.Comments) > 0 {
			result.Threads = append(result.Threads, thread)
		}
	}

	// Convert issue comments
	for _, c := range resp.Data.Repository.PullRequest.Comments.Nodes {
		createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
		result.IssueComments = append(result.IssueComments, IssueComment{
			ID:        c.DatabaseID,
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: createdAt,
		})
	}

	return result, nil
}

// ReplyToReviewThread posts a reply to a review comment thread.
func ReplyToReviewThread(dir string, prNumber int, commentID int, body string) error {
	owner, repo, err := RepoFromRemote(dir)
	if err != nil {
		return fmt.Errorf("resolving repo: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/comments/%d/replies", owner, repo, prNumber, commentID)
	cmd := exec.Command("gh", "api", "-X", "POST", endpoint, "-f", fmt.Sprintf("body=%s", body))
	cmd.Dir = dir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("posting reply: %s: %w", string(out), err)
	}

	return nil
}

// ReplyToIssue posts a top-level comment on the PR (issue comments endpoint).
func ReplyToIssue(dir string, prNumber int, body string) error {
	owner, repo, err := RepoFromRemote(dir)
	if err != nil {
		return fmt.Errorf("resolving repo: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, prNumber)
	cmd := exec.Command("gh", "api", "-X", "POST", endpoint, "-f", fmt.Sprintf("body=%s", body))
	cmd.Dir = dir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("posting comment: %s: %w", string(out), err)
	}

	return nil
}
