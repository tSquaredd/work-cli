package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
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
