package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// Message types for async data fetching.

// tasksLoadedMsg is sent when tasks have been refreshed.
type tasksLoadedMsg struct {
	tasks []service.TaskView
}

// diffLoadedMsg is sent when a diff has been fetched.
type diffLoadedMsg struct {
	taskName string
	dir      string
	diff     string
}

// actionResultMsg is sent after an action completes.
type actionResultMsg struct {
	message string
	isError bool
}

// tickMsg triggers periodic refresh.
type tickMsg struct{}

// prStatusLoadedMsg is sent when PR status polling completes.
type prStatusLoadedMsg struct {
	tasks []service.TaskView
}

// openBrowserMsg triggers opening a URL in the browser.
type openBrowserMsg struct {
	url string
}

// Command factories.

// loadTasks fetches tasks from the service in the background.
func loadTasks(svc *service.WorkService) tea.Cmd {
	return func() tea.Msg {
		return tasksLoadedMsg{tasks: svc.Tasks()}
	}
}

// loadDiff fetches the full diff for a worktree directory.
func loadDiff(taskName, dir string) tea.Cmd {
	return func() tea.Msg {
		diff := service.Diff(dir)
		if diff == "" {
			diff = "(no changes)"
		}
		return diffLoadedMsg{
			taskName: taskName,
			dir:      dir,
			diff:     diff,
		}
	}
}

// pollPRStatus refreshes PR data for all tasks with known PRs.
func pollPRStatus(enricher *service.PREnricher, tasks []service.TaskView) tea.Cmd {
	return func() tea.Msg {
		refreshed := enricher.RefreshPRStatus(tasks)
		return prStatusLoadedMsg{tasks: refreshed}
	}
}

// discoverPRs runs initial PR discovery for worktrees without known PRs.
func discoverPRs(enricher *service.PREnricher, tasks []service.TaskView) tea.Cmd {
	return func() tea.Msg {
		discovered := enricher.DiscoverPRs(tasks)
		return prStatusLoadedMsg{tasks: discovered}
	}
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		return openBrowserMsg{url: url}
	}
}

// standalonePRsLoadedMsg is sent when standalone PRs have been fetched.
type standalonePRsLoadedMsg struct {
	mine   []service.StandalonePR
	others []service.StandalonePR
	err    error
}

// prDiffLoadedMsg is sent when a PR diff has been fetched for the diff viewer.
type prDiffLoadedMsg struct {
	repoDir   string
	repoAlias string
	prNumber  int
	prTitle   string
	headSHA   string
	isMine    bool
	diff      string
	err       error
}

// reviewCommentPostedMsg is sent after a review comment is posted from the diff viewer.
type reviewCommentPostedMsg struct {
	err error
}

// loadStandalonePRs fetches standalone PRs across all workspace repos.
func loadStandalonePRs(svc *service.WorkService, tasks []service.TaskView) tea.Cmd {
	return func() tea.Msg {
		mine, others, err := svc.StandalonePRs(tasks)
		return standalonePRsLoadedMsg{mine: mine, others: others, err: err}
	}
}

// loadPRDiff fetches the diff for a specific PR. If headSHA is empty, it fetches it.
func loadPRDiff(repoDir string, prNumber int, repoAlias, prTitle, headSHA string, isMine bool) tea.Cmd {
	return func() tea.Msg {
		// Fetch headSHA if not provided
		if headSHA == "" {
			sha, err := github.GetPRHeadSHA(repoDir, prNumber)
			if err == nil {
				headSHA = sha
			}
		}

		diff, err := github.GetPRDiff(repoDir, prNumber)
		return prDiffLoadedMsg{
			repoDir:   repoDir,
			repoAlias: repoAlias,
			prNumber:  prNumber,
			prTitle:   prTitle,
			headSHA:   headSHA,
			isMine:    isMine,
			diff:      diff,
			err:       err,
		}
	}
}

// postReviewComment posts a review comment from the diff viewer.
func postReviewComment(repoDir string, prNumber int, commitID, filePath string, startLine, endLine int, side, body string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if startLine == endLine {
			err = github.CreateReviewComment(repoDir, prNumber, commitID, filePath, endLine, side, body)
		} else {
			err = github.CreateMultiLineReviewComment(repoDir, prNumber, commitID, filePath, startLine, endLine, side, body)
		}
		if err != nil {
			return reviewCommentPostedMsg{err: fmt.Errorf("posting comment: %w", err)}
		}
		return reviewCommentPostedMsg{}
	}
}

// commentsLoadedMsg is sent when PR comments have been fetched.
type commentsLoadedMsg struct {
	taskName  string
	repoAlias string
	prNumber  int
	dir       string
	comments  *github.PRComments
	err       error
}

// commentRepliedMsg is sent after a comment reply is posted.
type commentRepliedMsg struct {
	err error
}

// loadComments fetches PR review comments in the background.
func loadComments(taskName, repoAlias, dir string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		comments, err := github.FetchPRComments(dir, prNumber)
		return commentsLoadedMsg{
			taskName:  taskName,
			repoAlias: repoAlias,
			prNumber:  prNumber,
			dir:       dir,
			comments:  comments,
			err:       err,
		}
	}
}

// replyToComment posts a reply to a review thread comment.
func replyToComment(dir string, prNumber, commentID int, body string) tea.Cmd {
	return func() tea.Msg {
		err := github.ReplyToReviewThread(dir, prNumber, commentID, body)
		return commentRepliedMsg{err: err}
	}
}

// replyToIssueComment posts a top-level PR comment.
func replyToIssueComment(dir string, prNumber int, body string) tea.Cmd {
	return func() tea.Msg {
		err := github.ReplyToIssue(dir, prNumber, body)
		return commentRepliedMsg{err: err}
	}
}

// --- New task overlay messages ---

// newTaskFormDoneMsg signals that the current form step completed.
type newTaskFormDoneMsg struct{}

// newTaskFormCancelMsg signals that the form was cancelled (Esc).
type newTaskFormCancelMsg struct{}

// newTaskCreatedMsg is sent when worktree creation finishes.
type newTaskCreatedMsg struct {
	dirs     []string
	progress []string
	err      error
}

// createWorktrees runs fetch/create/link for each repo config in the background.
func createWorktrees(ws *workspace.Workspace, taskName string, configs []repoConfig) tea.Cmd {
	return func() tea.Msg {
		var dirs []string
		var progress []string

		for _, cfg := range configs {
			repo := ws.RepoByAlias(cfg.Alias)
			if repo == nil {
				continue
			}
			wtDir := worktree.WorktreeDir(ws, taskName, cfg.Alias)

			// Fix 2: switch main repo branch so worktree add can use the PR branch.
			if cfg.SwitchTo != "" {
				if err := worktree.Checkout(repo.Path, cfg.SwitchTo); err != nil {
					progress = append(progress, fmt.Sprintf("%s: failed to switch to %s — %s", cfg.Alias, cfg.SwitchTo, err))
					continue
				}
				progress = append(progress, fmt.Sprintf("%s: switched to %s", cfg.Alias, cfg.SwitchTo))
			}

			// Fetch
			progress = append(progress, fmt.Sprintf("%s: fetching origin/%s...", cfg.Alias, cfg.BaseBranch))
			_ = worktree.Fetch(repo.Path, cfg.BaseBranch)

			// Create
			result := worktree.Create(worktree.CreateConfig{
				RepoDir:     repo.Path,
				WorktreeDir: wtDir,
				Branch:      cfg.Branch,
				BaseBranch:  cfg.BaseBranch,
			})

			if result.Error != nil {
				progress = append(progress, fmt.Sprintf("%s: failed — %s", cfg.Alias, result.Error))
				continue
			}

			if !result.Created {
				progress = append(progress, fmt.Sprintf("%s: worktree already exists", cfg.Alias))
			} else if result.Attached {
				progress = append(progress, fmt.Sprintf("%s: attached to existing branch %s", cfg.Alias, cfg.Branch))
			} else {
				progress = append(progress, fmt.Sprintf("%s: created (%s from origin/%s)", cfg.Alias, cfg.Branch, cfg.BaseBranch))
			}

			// Link build files
			linkResult := worktree.LinkBuildFiles(result.Dir, repo.Path)
			if len(linkResult.Files) > 0 {
				progress = append(progress, fmt.Sprintf("%s: linked %s", cfg.Alias, strings.Join(linkResult.Files, ", ")))
			}

			dirs = append(dirs, result.Dir)
		}

		if len(dirs) == 0 {
			return newTaskCreatedMsg{err: fmt.Errorf("no worktrees created")}
		}
		return newTaskCreatedMsg{dirs: dirs, progress: progress}
	}
}
