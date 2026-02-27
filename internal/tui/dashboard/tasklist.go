package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/ui"
)

type rowKind int

const (
	rowTask rowKind = iota
	rowWorktree
	rowSectionHeader
	rowMyPR
	rowOtherPR
)

type listRow struct {
	kind    rowKind
	taskIdx int // index into tasks
	wtIdx   int // index into task.Worktrees
	prIdx   int // index into myPRs or otherPRs
}

// taskListModel manages the left panel — task list with cursor and session indicators.
type taskListModel struct {
	tasks    []service.TaskView
	myPRs    []service.StandalonePR
	otherPRs []service.StandalonePR
	rows     []listRow   // computed flat list
	cursor   int         // index into rows
	expanded map[int]bool // tracks which tasks have their worktree list expanded
	width    int
	height   int
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
	m.buildRows()
}

func (m *taskListModel) setStandalonePRs(mine, others []service.StandalonePR) {
	m.myPRs = mine
	m.otherPRs = others
	m.buildRows()
}

// buildRows recomputes the flat row list from tasks + standalone PRs.
func (m *taskListModel) buildRows() {
	var rows []listRow

	for i, t := range m.tasks {
		rows = append(rows, listRow{kind: rowTask, taskIdx: i})
		if m.expanded[i] {
			for j := range t.Worktrees {
				rows = append(rows, listRow{kind: rowWorktree, taskIdx: i, wtIdx: j})
			}
		}
	}

	if len(m.myPRs) > 0 {
		rows = append(rows, listRow{kind: rowSectionHeader, prIdx: -1}) // "Your PRs" header
		for i := range m.myPRs {
			rows = append(rows, listRow{kind: rowMyPR, prIdx: i})
		}
	}

	if len(m.otherPRs) > 0 {
		rows = append(rows, listRow{kind: rowSectionHeader, prIdx: -2}) // "Reviews" header
		for i := range m.otherPRs {
			rows = append(rows, listRow{kind: rowOtherPR, prIdx: i})
		}
	}

	m.rows = rows

	// Clamp cursor
	if len(rows) > 0 {
		if m.cursor >= len(rows) {
			m.cursor = len(rows) - 1
		}
		// Skip section headers
		m.skipHeaders(1)
	}
}

// selected returns the currently selected task, or nil if cursor is on a non-task row.
func (m *taskListModel) selected() *service.TaskView {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	row := m.rows[m.cursor]
	switch row.kind {
	case rowTask, rowWorktree:
		if row.taskIdx >= 0 && row.taskIdx < len(m.tasks) {
			return &m.tasks[row.taskIdx]
		}
	}
	return nil
}

// selectedRow returns the row at cursor position.
func (m *taskListModel) selectedRow() *listRow {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return &m.rows[m.cursor]
	}
	return nil
}

// selectedStandalonePR returns the standalone PR at cursor, or nil.
func (m *taskListModel) selectedStandalonePR() *service.StandalonePR {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	row := m.rows[m.cursor]
	switch row.kind {
	case rowMyPR:
		if row.prIdx >= 0 && row.prIdx < len(m.myPRs) {
			return &m.myPRs[row.prIdx]
		}
	case rowOtherPR:
		if row.prIdx >= 0 && row.prIdx < len(m.otherPRs) {
			return &m.otherPRs[row.prIdx]
		}
	}
	return nil
}

func (m *taskListModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.skipHeaders(-1)
	}
}

func (m *taskListModel) moveDown() {
	if m.cursor < len(m.rows)-1 {
		m.cursor++
		m.skipHeaders(1)
	}
}

// skipHeaders advances cursor past section headers in the given direction.
func (m *taskListModel) skipHeaders(dir int) {
	for m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowSectionHeader {
		m.cursor += dir
	}
	// Clamp
	if m.cursor < 0 {
		m.cursor = 0
		// If first row is a header, skip forward
		if len(m.rows) > 0 && m.rows[0].kind == rowSectionHeader {
			m.cursor = 1
		}
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
}

func (m *taskListModel) toggleExpand() {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	row := m.rows[m.cursor]
	if row.kind == rowTask {
		m.expanded[row.taskIdx] = !m.expanded[row.taskIdx]
		m.buildRows()
	}
}

func (m taskListModel) view() string {
	if len(m.rows) == 0 {
		return ui.StyleDim.Render("  No active tasks.")
	}

	var b strings.Builder

	for i, row := range m.rows {
		isCursor := i == m.cursor

		switch row.kind {
		case rowTask:
			b.WriteString(m.renderTaskRow(row, isCursor))
		case rowWorktree:
			b.WriteString(m.renderWorktreeRow(row, isCursor))
		case rowSectionHeader:
			b.WriteString(m.renderSectionHeader(row))
		case rowMyPR:
			b.WriteString(m.renderMyPRRow(row, isCursor))
		case rowOtherPR:
			b.WriteString(m.renderOtherPRRow(row, isCursor))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m taskListModel) renderTaskRow(row listRow, isCursor bool) string {
	t := m.tasks[row.taskIdx]

	prefix := "  "
	if isCursor {
		prefix = ui.StylePrimary.Render("> ")
	}

	sessionMark := ""
	if t.HasSession {
		sessionMark = ui.StyleSuccess.Render(" *")
	}

	name := ui.StyleTaskName.Render(t.Name)
	return prefix + name + sessionMark
}

func (m taskListModel) renderWorktreeRow(row listRow, isCursor bool) string {
	t := m.tasks[row.taskIdx]
	wt := t.Worktrees[row.wtIdx]
	isLast := row.wtIdx == len(t.Worktrees)-1

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	repoName := ui.StyleRepoName.Render(padRight(wt.Alias, 16))
	badge := ui.StatusBadge(wt.Status.String())

	// PR indicator
	prIndicator := ""
	if wt.PR != nil && wt.PR.Number > 0 {
		prBadge := ui.PRBadge(wt.PR.State, wt.PR.ReviewStatus)
		prNum := ui.StyleDim.Render(fmt.Sprintf("#%d", wt.PR.Number))
		prIndicator = fmt.Sprintf("  %s %s", prBadge, prNum)
		if wt.PR.NewComments > 0 {
			prIndicator += ui.StyleWarning.Render(fmt.Sprintf(" (%d)", wt.PR.NewComments))
		}
	}

	pad := "  "
	if isCursor {
		pad = "  "
	}

	return fmt.Sprintf("  %s%s%s  %s%s",
		pad,
		ui.StyleTreeBranch.Render(connector),
		repoName,
		badge,
		prIndicator,
	)
}

func (m taskListModel) renderSectionHeader(row listRow) string {
	var label string
	if row.prIdx == -1 {
		label = "Your PRs"
	} else {
		label = "Reviews"
	}

	headerStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Faint(true)

	maxWidth := m.width - 4
	if maxWidth < 20 {
		maxWidth = 30
	}

	// Render: ── Label ──────────
	prefix := "── "
	suffix := " "
	remaining := maxWidth - len(prefix) - len(label) - len(suffix)
	if remaining < 3 {
		remaining = 3
	}
	line := prefix + label + suffix + strings.Repeat("─", remaining)

	return "\n" + headerStyle.Render(line)
}

func (m taskListModel) renderMyPRRow(row listRow, isCursor bool) string {
	pr := m.myPRs[row.prIdx]
	return m.renderPRRow(pr, isCursor, false)
}

func (m taskListModel) renderOtherPRRow(row listRow, isCursor bool) string {
	pr := m.otherPRs[row.prIdx]
	return m.renderPRRow(pr, isCursor, true)
}

func (m taskListModel) renderPRRow(pr service.StandalonePR, isCursor bool, showAuthor bool) string {
	prefix := "  "
	if isCursor {
		prefix = ui.StylePrimary.Render("> ")
	}

	repoName := ui.StyleRepoName.Render(padRight(pr.RepoAlias, 12))
	prBadge := ui.PRBadge("OPEN", pr.ReviewStatus)
	prNum := ui.StyleDim.Render(fmt.Sprintf("#%d", pr.Number))

	// Truncate title to fit
	title := pr.Title
	maxTitle := m.width - 28
	if showAuthor {
		maxTitle -= 12
	}
	if maxTitle < 10 {
		maxTitle = 10
	}
	if len(title) > maxTitle {
		title = title[:maxTitle-3] + "..."
	}

	line := fmt.Sprintf("%s%s  %s %s  %s", prefix, repoName, prBadge, prNum, title)

	if showAuthor {
		author := ui.StyleDim.Render(fmt.Sprintf("@%s", pr.Author))
		line += "  " + author
	}

	return line
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

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
