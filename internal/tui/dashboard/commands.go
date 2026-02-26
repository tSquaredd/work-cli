package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/service"
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
