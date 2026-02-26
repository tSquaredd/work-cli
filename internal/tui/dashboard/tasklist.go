package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/ui"
)

// taskListModel manages the left panel — task list with cursor and session indicators.
type taskListModel struct {
	tasks    []service.TaskView
	cursor   int
	expanded map[int]bool // tracks which tasks have their worktree list expanded
	width    int
	height   int
	filter   string
}

func newTaskListModel() taskListModel {
	return taskListModel{
		expanded: make(map[int]bool),
	}
}

func (m *taskListModel) setTasks(tasks []service.TaskView) {
	m.tasks = tasks
	// Expand all by default
	for i := range tasks {
		if _, ok := m.expanded[i]; !ok {
			m.expanded[i] = true
		}
	}
	// Clamp cursor
	if m.cursor >= len(tasks) && len(tasks) > 0 {
		m.cursor = len(tasks) - 1
	}
}

func (m *taskListModel) filteredTasks() []service.TaskView {
	if m.filter == "" {
		return m.tasks
	}
	var filtered []service.TaskView
	lower := strings.ToLower(m.filter)
	for _, t := range m.tasks {
		if strings.Contains(strings.ToLower(t.Name), lower) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (m *taskListModel) selected() *service.TaskView {
	tasks := m.filteredTasks()
	if m.cursor >= 0 && m.cursor < len(tasks) {
		return &tasks[m.cursor]
	}
	return nil
}

func (m *taskListModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *taskListModel) moveDown() {
	tasks := m.filteredTasks()
	if m.cursor < len(tasks)-1 {
		m.cursor++
	}
}

func (m *taskListModel) toggleExpand() {
	m.expanded[m.cursor] = !m.expanded[m.cursor]
}

func (m taskListModel) view() string {
	tasks := m.filteredTasks()
	if len(tasks) == 0 {
		msg := "No active tasks."
		if m.filter != "" {
			msg = fmt.Sprintf("No tasks matching %q", m.filter)
		}
		return ui.StyleDim.Render("  " + msg)
	}

	var b strings.Builder

	for i, t := range tasks {
		isCursor := i == m.cursor

		// Task name with session indicator
		prefix := "  "
		if isCursor {
			prefix = ui.StylePrimary.Render("> ")
		}

		sessionMark := ""
		if t.HasSession {
			sessionMark = ui.StyleSuccess.Render(" *")
		}

		name := ui.StyleTaskName.Render(t.Name)
		b.WriteString(prefix + name + sessionMark + "\n")

		// Worktree list (if expanded)
		if m.expanded[i] {
			for j, wt := range t.Worktrees {
				isLast := j == len(t.Worktrees)-1
				connector := "├── "
				if isLast {
					connector = "└── "
				}

				repoName := ui.StyleRepoName.Render(padRight(wt.Alias, 16))
				badge := ui.StatusBadge(wt.Status.String())

				line := fmt.Sprintf("  %s%s%s  %s",
					prefix_pad(isCursor),
					ui.StyleTreeBranch.Render(connector),
					repoName,
					badge,
				)
				b.WriteString(line + "\n")
			}
		}
	}

	return b.String()
}

func prefix_pad(isCursor bool) string {
	if isCursor {
		return "  "
	}
	return "  "
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// countActive returns the number of tasks with active sessions.
func countActive(tasks []service.TaskView) int {
	n := 0
	for _, t := range tasks {
		if t.HasSession {
			n++
		}
	}
	return n
}

// headerLine renders the dashboard header with task/session counts.
func headerLine(tasks []service.TaskView, width int) string {
	title := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		Render("work dashboard")

	active := countActive(tasks)
	stats := ui.StyleDim.Render(fmt.Sprintf("%d tasks  %d active", len(tasks), active))

	gap := width - lipgloss.Width(title) - lipgloss.Width(stats)
	if gap < 2 {
		gap = 2
	}

	return title + strings.Repeat(" ", gap) + stats
}
