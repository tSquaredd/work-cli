package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/ui"
)

// detailModel manages the right panel — worktree detail and diff rendering.
type detailModel struct {
	task      *service.TaskView
	diffText  string     // full diff when loaded
	diffDir   string     // which worktree dir the diff is for
	showDiff  bool       // whether to show full diff
	scroll    int        // scroll offset for diff view
	width     int
	height    int
}

func newDetailModel() detailModel {
	return detailModel{}
}

func (m *detailModel) setTask(task *service.TaskView) {
	if m.task == nil || task == nil || m.task.Name != task.Name {
		m.diffText = ""
		m.diffDir = ""
		m.showDiff = false
		m.scroll = 0
	}
	m.task = task
}

func (m *detailModel) scrollUp() {
	if m.scroll > 0 {
		m.scroll--
	}
}

func (m *detailModel) scrollDown() {
	m.scroll++
}

func (m detailModel) view() string {
	if m.task == nil {
		return ui.StyleDim.Render("  Select a task to view details")
	}

	var b strings.Builder

	// Task title
	title := lipgloss.NewStyle().
		Foreground(ui.ColorInfo).
		Bold(true).
		Render(m.task.Name)
	b.WriteString(title + "\n\n")

	if m.showDiff && m.diffText != "" {
		b.WriteString(m.renderDiff())
		return b.String()
	}

	// Worktree details
	for _, wt := range m.task.Worktrees {
		repoName := ui.StyleRepoName.Render(wt.Alias)
		badge := ui.StatusBadge(wt.Status.String())
		b.WriteString(fmt.Sprintf("%s  %s\n", repoName, badge))

		// Branch
		branch := ui.StyleBranchName.Render(fmt.Sprintf("  branch: %s", wt.Branch))
		b.WriteString(branch + "\n")

		// Changed files
		if wt.DiffStat != "" {
			b.WriteString(ui.StyleDim.Render(fmt.Sprintf("  %s", wt.DiffStat)) + "\n")
		} else {
			b.WriteString(ui.StyleDim.Render("  (clean)") + "\n")
		}

		b.WriteString("\n")
	}

	// Session info
	if m.task.HasSession {
		sessionStyle := lipgloss.NewStyle().
			Foreground(ui.ColorSuccess).
			Bold(true)
		b.WriteString(sessionStyle.Render("Session: ACTIVE") + "\n")

		elapsed := time.Since(m.task.SessionLaunchedAt).Truncate(time.Second)
		b.WriteString(ui.StyleDim.Render(
			fmt.Sprintf("  PID %d · %s ago", m.task.SessionPID, formatDuration(elapsed)),
		) + "\n")
	}

	return b.String()
}

func (m detailModel) renderDiff() string {
	lines := strings.Split(m.diffText, "\n")

	// Apply scroll offset
	start := m.scroll
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}

	maxLines := m.height - 4 // leave room for header
	if maxLines < 1 {
		maxLines = 20
	}

	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	addStyle := lipgloss.NewStyle().Foreground(ui.ColorSuccess)
	delStyle := lipgloss.NewStyle().Foreground(ui.ColorDanger)
	hunkStyle := lipgloss.NewStyle().Foreground(ui.ColorInfo)

	for _, line := range lines[start:end] {
		// Truncate long lines to panel width
		displayLine := line
		if m.width > 0 && len(displayLine) > m.width-2 {
			displayLine = displayLine[:m.width-5] + "..."
		}

		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			b.WriteString(addStyle.Render(displayLine))
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			b.WriteString(delStyle.Render(displayLine))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(hunkStyle.Render(displayLine))
		default:
			b.WriteString(displayLine)
		}
		b.WriteString("\n")
	}

	// Scroll indicator
	if end < len(lines) {
		b.WriteString(ui.StyleDim.Render(
			fmt.Sprintf("  ... %d more lines (j/k to scroll)", len(lines)-end),
		))
	}

	return b.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
