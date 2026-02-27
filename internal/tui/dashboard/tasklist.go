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
	rowRepoHeader
	rowMyPR
	rowOtherPR
	rowSectionMessage
)

type listRow struct {
	kind      rowKind
	taskIdx   int    // index into tasks
	wtIdx     int    // index into task.Worktrees
	prIdx     int    // index into myPRs or otherPRs
	repoAlias string // for rowRepoHeader: the repo alias
	section   int    // for rowRepoHeader: 1 = "Your PRs", 2 = "Reviews"
}

type navLevel int

const (
	navGroup navLevel = iota // cursor on a task/PR group row
	navRepo                  // cursor inside a task's worktree list
)

// taskListModel manages the left panel — task list with cursor and session indicators.
type taskListModel struct {
	tasks     []service.TaskView
	myPRs     []service.StandalonePR
	otherPRs  []service.StandalonePR
	rows      []listRow        // computed flat list
	cursor    int              // index into rows (always a group-level row)
	navLevel  navLevel         // whether cursor is at group or repo level
	wtCursor  int              // focused worktree index when at repo level
	collapsed map[string]bool  // key: "section:repoAlias", true if collapsed
	prLoaded  bool            // true after first standalone PR fetch completes
	viewTop   int              // first visible row index for viewport scrolling
	width     int
	height    int
	prError   string // persistent error message from PR list fetch
}

func newTaskListModel() taskListModel {
	return taskListModel{
		collapsed: make(map[string]bool),
	}
}

func (m *taskListModel) setTasks(tasks []service.TaskView) {
	m.tasks = tasks
	m.buildRows()
}

func (m *taskListModel) setStandalonePRs(mine, others []service.StandalonePR) {
	m.myPRs = mine
	m.otherPRs = others
	m.prLoaded = true
	m.prError = "" // clear error on successful load
	m.buildRows()
}

func (m *taskListModel) setPRError(msg string) {
	m.prError = msg
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

	// Always show both PR sections
	rows = m.appendPRSection(rows, m.myPRs, rowMyPR, 1, -1)  // section 1 = "Your PRs"
	rows = m.appendPRSection(rows, m.otherPRs, rowOtherPR, 2, -2) // section 2 = "Not Your PRs"

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

		m.ensureVisible()
	}
}

// appendPRSection always emits a section header and either grouped PR rows,
// a loading message, or an empty message.
func (m *taskListModel) appendPRSection(rows []listRow, prs []service.StandalonePR, kind rowKind, section int, sectionIdx int) []listRow {
	rows = append(rows, listRow{kind: rowSectionHeader, prIdx: sectionIdx})

	if len(prs) == 0 {
		// Show loading or empty message
		rows = append(rows, listRow{kind: rowSectionMessage, section: section})
		return rows
	}

	return m.appendGroupedPRRows(rows, prs, kind, section)
}

// appendGroupedPRRows groups PRs by repo alias (preserving first-appearance order)
// and emits repo headers with child PR rows.
func (m *taskListModel) appendGroupedPRRows(rows []listRow, prs []service.StandalonePR, kind rowKind, section int) []listRow {

	// Group by repo, preserving first-appearance order
	type repoGroup struct {
		alias   string
		indices []int
	}
	var groups []repoGroup
	repoMap := map[string]int{} // alias -> index in groups
	for i, pr := range prs {
		idx, ok := repoMap[pr.RepoAlias]
		if !ok {
			repoMap[pr.RepoAlias] = len(groups)
			groups = append(groups, repoGroup{alias: pr.RepoAlias, indices: []int{i}})
		} else {
			groups[idx].indices = append(groups[idx].indices, i)
		}
	}

	for _, g := range groups {
		rows = append(rows, listRow{
			kind:      rowRepoHeader,
			repoAlias: g.alias,
			section:   section,
		})

		collapseKey := fmt.Sprintf("%d:%s", section, g.alias)
		if _, seen := m.collapsed[collapseKey]; !seen {
			m.collapsed[collapseKey] = true // default to collapsed
		}
		if !m.collapsed[collapseKey] {
			for _, prIdx := range g.indices {
				rows = append(rows, listRow{kind: kind, prIdx: prIdx})
			}
		}
	}

	return rows
}

// collapseKey returns the map key for a repo header's collapse state.
func collapseKey(section int, repoAlias string) string {
	return fmt.Sprintf("%d:%s", section, repoAlias)
}

// expandRepoHeader expands the repo header at cursor if it is collapsed.
func (m *taskListModel) expandRepoHeader() {
	row := m.selectedRow()
	if row == nil || row.kind != rowRepoHeader {
		return
	}
	key := collapseKey(row.section, row.repoAlias)
	if m.collapsed[key] {
		m.collapsed[key] = false
		m.buildRows() // buildRows calls ensureVisible
	}
}

// collapseRepoHeader collapses the repo header at cursor if it is expanded.
func (m *taskListModel) collapseRepoHeader() {
	row := m.selectedRow()
	if row == nil || row.kind != rowRepoHeader {
		return
	}
	key := collapseKey(row.section, row.repoAlias)
	if !m.collapsed[key] {
		m.collapsed[key] = true
		m.buildRows()
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
		m.ensureVisible()
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
		m.ensureVisible()
	}
}

// skipToGroupRow advances cursor past section headers and worktree rows in the given direction.
func (m *taskListModel) skipToGroupRow(dir int) {
	for m.cursor >= 0 && m.cursor < len(m.rows) {
		k := m.rows[m.cursor].kind
		if k != rowSectionHeader && k != rowSectionMessage && k != rowWorktree {
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
			if k != rowSectionHeader && k != rowSectionMessage && k != rowWorktree {
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
			if k != rowSectionHeader && k != rowSectionMessage && k != rowWorktree {
				break
			}
			m.cursor--
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

// ensureVisible adjusts viewTop so the cursor row is within the visible viewport.
func (m *taskListModel) ensureVisible() {
	if m.height <= 0 {
		return
	}
	// Scroll up if cursor is above viewport
	if m.cursor < m.viewTop {
		m.viewTop = m.cursor
	}
	// Scroll down if cursor is below viewport
	for {
		vis := m.visibleLines(m.viewTop, m.cursor)
		if vis <= m.height {
			break
		}
		m.viewTop++
	}
}

// visibleLines counts the visual lines occupied by rows[from..to] inclusive.
// Section headers take 2 lines (they render with a leading \n).
func (m *taskListModel) visibleLines(from, to int) int {
	n := 0
	for i := from; i <= to && i < len(m.rows); i++ {
		n++
		if m.rows[i].kind == rowSectionHeader {
			n++ // section headers render with leading \n
		}
	}
	return n
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
	visLines := 0

	for i := m.viewTop; i < len(m.rows); i++ {
		row := m.rows[i]

		// Check if this row fits in the viewport
		rowLines := 1
		if row.kind == rowSectionHeader {
			rowLines = 2 // section headers render with leading \n
		}
		if m.height > 0 && visLines+rowLines > m.height {
			break
		}

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
		case rowSectionMessage:
			b.WriteString(m.renderSectionMessage(row))
		case rowRepoHeader:
			b.WriteString(m.renderRepoHeader(row, isGroupCursor))
		case rowMyPR:
			b.WriteString(m.renderMyPRRow(row, isGroupCursor))
		case rowOtherPR:
			b.WriteString(m.renderOtherPRRow(row, isGroupCursor))
		}
		b.WriteString("\n")
		visLines += rowLines
	}

	// Show PR error if no standalone PRs loaded (only when visible at bottom)
	if m.prError != "" && len(m.myPRs) == 0 && len(m.otherPRs) == 0 {
		if m.height <= 0 || visLines+2 <= m.height {
			errText := m.prError
			maxW := m.width - 6 // account for "  ⚠ " prefix
			if maxW > 0 && len(errText) > maxW {
				errText = errText[:maxW-3] + "..."
			}
			b.WriteString("\n")
			b.WriteString(ui.StyleDim.Render("  ⚠ " + errText))
			b.WriteString("\n")
		}
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
		prBadge := ui.PRBadge(wt.PR.State, wt.PR.ReviewStatus, false)
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

func (m taskListModel) renderSectionMessage(row listRow) string {
	var msg string
	if !m.prLoaded {
		msg = "Loading..."
	} else {
		msg = "None"
	}
	return "  " + ui.StyleDim.Render(msg)
}

func (m taskListModel) renderSectionHeader(row listRow) string {
	var label string
	if row.prIdx == -1 {
		label = "Your PRs"
	} else {
		label = "Not Your PRs"
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

func (m taskListModel) renderRepoHeader(row listRow, isCursor bool) string {
	prefix := "  "
	if isCursor {
		prefix = ui.StylePrimary.Render("> ")
	}

	key := collapseKey(row.section, row.repoAlias)
	collapsed := m.collapsed[key]

	indicator := ui.StyleDim.Render("▾")
	if collapsed {
		indicator = ui.StyleDim.Render("▸")
	}

	repoName := ui.StyleRepoName.Render(row.repoAlias)

	// Count PRs for this repo in the appropriate section
	var prs []service.StandalonePR
	if row.section == 1 {
		prs = m.myPRs
	} else {
		prs = m.otherPRs
	}
	count := 0
	for _, pr := range prs {
		if pr.RepoAlias == row.repoAlias {
			count++
		}
	}
	countStr := ui.StyleDim.Render(fmt.Sprintf("(%d)", count))

	return fmt.Sprintf("%s%s %s %s", prefix, indicator, repoName, countStr)
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
	prefix := "    "
	if isCursor {
		prefix = "  " + ui.StylePrimary.Render("> ")
	}

	prBadge := ui.PRBadge("OPEN", pr.ReviewStatus, pr.IsDraft)
	prNum := ui.StyleDim.Render(fmt.Sprintf("#%-5d", pr.Number))

	// Build author suffix first so we know its visual width
	var authorSuffix string
	if showAuthor {
		authorSuffix = "  " + ui.StyleDim.Render(fmt.Sprintf("@%s", pr.Author))
	}

	// Measure fixed parts visually (ANSI-aware) to get accurate remaining space
	fixedWidth := lipgloss.Width(prefix) + lipgloss.Width(prBadge) + 1 +
		lipgloss.Width(prNum) + 1 + lipgloss.Width(authorSuffix)
	maxTitle := m.width - fixedWidth
	if maxTitle < 10 {
		maxTitle = 10
	}

	title := pr.Title
	if len(title) > maxTitle {
		title = title[:maxTitle-3] + "..."
	}

	return fmt.Sprintf("%s%s %s %s", prefix, prBadge, prNum, title) + authorSuffix
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
