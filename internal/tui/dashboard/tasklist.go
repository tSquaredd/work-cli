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

type navLevel int

const (
	navGroup navLevel = iota // cursor on a task/PR group row
	navRepo                  // cursor inside a task's worktree list
)

// taskListModel manages the left panel — task list with cursor and session indicators.
type taskListModel struct {
	tasks    []service.TaskView
	myPRs    []service.StandalonePR
	otherPRs []service.StandalonePR
	rows     []listRow  // computed flat list
	cursor   int        // index into rows (always a group-level row)
	navLevel navLevel   // whether cursor is at group or repo level
	wtCursor int        // focused worktree index when at repo level
	width    int
	height   int
}

func newTaskListModel() taskListModel {
	return taskListModel{}
}

func (m *taskListModel) setTasks(tasks []service.TaskView) {
	m.tasks = tasks
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
		for j := range t.Worktrees {
			rows = append(rows, listRow{kind: rowWorktree, taskIdx: i, wtIdx: j})
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

	// Clamp cursor to a valid group row
	if len(rows) > 0 {
		if m.cursor >= len(rows) {
			m.cursor = len(rows) - 1
		}
		m.skipToGroupRow(1)

		// Clamp wtCursor if at repo level
		if m.navLevel == navRepo {
			if row := m.selectedRow(); row != nil && row.kind == rowTask {
				wts := m.tasks[row.taskIdx].Worktrees
				if len(wts) == 0 {
					m.navLevel = navGroup
					m.wtCursor = 0
				} else if m.wtCursor >= len(wts) {
					m.wtCursor = len(wts) - 1
				}
			} else {
				m.navLevel = navGroup
				m.wtCursor = 0
			}
		}
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
	if m.navLevel == navRepo {
		// Move between worktrees within the current task
		if m.wtCursor > 0 {
			m.wtCursor--
		}
		return
	}
	if m.cursor > 0 {
		m.cursor--
		m.skipToGroupRow(-1)
	}
}

func (m *taskListModel) moveDown() {
	if m.navLevel == navRepo {
		// Move between worktrees within the current task
		row := m.selectedRow()
		if row != nil && row.kind == rowTask {
			wts := m.tasks[row.taskIdx].Worktrees
			if m.wtCursor < len(wts)-1 {
				m.wtCursor++
			}
		}
		return
	}
	if m.cursor < len(m.rows)-1 {
		m.cursor++
		m.skipToGroupRow(1)
	}
}

// skipToGroupRow advances cursor past section headers and worktree rows in the given direction.
func (m *taskListModel) skipToGroupRow(dir int) {
	for m.cursor >= 0 && m.cursor < len(m.rows) {
		k := m.rows[m.cursor].kind
		if k != rowSectionHeader && k != rowWorktree {
			break
		}
		m.cursor += dir
	}
	// Clamp
	if m.cursor < 0 {
		m.cursor = 0
		// If first row is skippable, find next valid row
		for m.cursor < len(m.rows) {
			k := m.rows[m.cursor].kind
			if k != rowSectionHeader && k != rowWorktree {
				break
			}
			m.cursor++
		}
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
		// Walk back if we landed on a skippable row
		for m.cursor >= 0 {
			k := m.rows[m.cursor].kind
			if k != rowSectionHeader && k != rowWorktree {
				break
			}
			m.cursor--
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

// enterWorktrees switches to repo-level navigation within the current task.
func (m *taskListModel) enterWorktrees() {
	row := m.selectedRow()
	if row == nil || row.kind != rowTask {
		return
	}
	if len(m.tasks[row.taskIdx].Worktrees) == 0 {
		return
	}
	m.navLevel = navRepo
	m.wtCursor = 0
}

// exitWorktrees returns to group-level navigation.
func (m *taskListModel) exitWorktrees() {
	m.navLevel = navGroup
	m.wtCursor = 0
}

// focusedWorktree returns the specific worktree when at repo level, nil otherwise.
func (m *taskListModel) focusedWorktree() *service.WorktreeView {
	if m.navLevel != navRepo {
		return nil
	}
	row := m.selectedRow()
	if row == nil || row.kind != rowTask {
		return nil
	}
	wts := m.tasks[row.taskIdx].Worktrees
	if m.wtCursor < 0 || m.wtCursor >= len(wts) {
		return nil
	}
	return &wts[m.wtCursor]
}

func (m taskListModel) view() string {
	if len(m.rows) == 0 {
		return ui.StyleDim.Render("  No active tasks.")
	}

	var b strings.Builder

	for i, row := range m.rows {
		isGroupCursor := i == m.cursor && m.navLevel == navGroup

		switch row.kind {
		case rowTask:
			b.WriteString(m.renderTaskRow(row, isGroupCursor))
		case rowWorktree:
			// Show worktree cursor when at repo level and this worktree matches wtCursor
			isWtCursor := m.navLevel == navRepo && m.rows[m.cursor].kind == rowTask &&
				row.taskIdx == m.rows[m.cursor].taskIdx && row.wtIdx == m.wtCursor
			b.WriteString(m.renderWorktreeRow(row, isWtCursor))
		case rowSectionHeader:
			b.WriteString(m.renderSectionHeader(row))
		case rowMyPR:
			b.WriteString(m.renderMyPRRow(row, isGroupCursor))
		case rowOtherPR:
			b.WriteString(m.renderOtherPRRow(row, isGroupCursor))
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

	pad := "    "
	if isCursor {
		pad = "  " + ui.StylePrimary.Render("> ")
	}

	return fmt.Sprintf("%s%s%s  %s%s",
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
