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
