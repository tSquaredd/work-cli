package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/ui"
)

type commentMode int

const (
	commentModeBrowse commentMode = iota
	commentModeReply
	commentModeClaudePrompt
)

// commentViewModel manages the fullscreen comment viewer overlay.
type commentViewModel struct {
	// Data
	taskName    string
	repoAlias   string
	prNumber    int
	worktreeDir string
	threads     []github.ReviewThread
	issues      []github.IssueComment

	// Navigation
	currentIdx int         // index into combined list (threads then issues)
	mode       commentMode // browse or reply

	// Reply
	replyBuf []rune
	replyErr string

	// Claude prompt editing
	claudePromptBuf []rune

	// Layout
	width, height int
	scroll        int

	// Exit signals
	claudeRequested bool
	claudeThread    *github.ReviewThread
}

func newCommentViewModel() commentViewModel {
	return commentViewModel{}
}

func (m *commentViewModel) setData(taskName, repoAlias string, prNumber int, dir string, comments *github.PRComments) {
	m.taskName = taskName
	m.repoAlias = repoAlias
	m.prNumber = prNumber
	m.worktreeDir = dir
	m.threads = comments.Threads
	m.issues = comments.IssueComments
	m.currentIdx = 0
	m.scroll = 0
	m.mode = commentModeBrowse
	m.replyBuf = nil
	m.replyErr = ""
	m.claudePromptBuf = nil
	m.claudeRequested = false
	m.claudeThread = nil
}

// totalItems returns the combined count of threads + issue comments.
func (m *commentViewModel) totalItems() int {
	return len(m.threads) + len(m.issues)
}

func (m *commentViewModel) next() {
	if m.currentIdx < m.totalItems()-1 {
		m.currentIdx++
		m.scroll = 0
	}
}

func (m *commentViewModel) prev() {
	if m.currentIdx > 0 {
		m.currentIdx--
		m.scroll = 0
	}
}

func (m *commentViewModel) scrollUp() {
	if m.scroll > 0 {
		m.scroll--
	}
}

func (m *commentViewModel) scrollDown() {
	m.scroll++
}

// isOnThread returns true if currentIdx points to a review thread.
func (m *commentViewModel) isOnThread() bool {
	return m.currentIdx < len(m.threads)
}

// currentThread returns the current thread, or nil if on an issue comment.
func (m *commentViewModel) currentThread() *github.ReviewThread {
	if m.isOnThread() {
		return &m.threads[m.currentIdx]
	}
	return nil
}

// currentIssueComment returns the current issue comment, or nil if on a thread.
func (m *commentViewModel) currentIssueComment() *github.IssueComment {
	idx := m.currentIdx - len(m.threads)
	if idx >= 0 && idx < len(m.issues) {
		return &m.issues[idx]
	}
	return nil
}

// lastCommentID returns the database ID of the last comment in the current thread (for replies).
func (m *commentViewModel) lastCommentID() int {
	if t := m.currentThread(); t != nil && len(t.Comments) > 0 {
		return t.Comments[len(t.Comments)-1].ID
	}
	return 0
}

func (m *commentViewModel) startReply() {
	m.mode = commentModeReply
	m.replyBuf = nil
	m.replyErr = ""
}

func (m *commentViewModel) cancelReply() {
	m.mode = commentModeBrowse
	m.replyBuf = nil
	m.replyErr = ""
}

func (m *commentViewModel) startClaudePrompt() {
	m.mode = commentModeClaudePrompt
	m.claudePromptBuf = nil
}

func (m *commentViewModel) cancelClaudePrompt() {
	m.mode = commentModeBrowse
	m.claudePromptBuf = nil
}

// claudeUserPrompt returns the user's additional instructions for Claude.
func (m *commentViewModel) claudeUserPrompt() string {
	return strings.TrimSpace(string(m.claudePromptBuf))
}

func (m commentViewModel) view() string {
	if m.totalItems() == 0 {
		return m.emptyView()
	}

	// Claude prompt editing gets its own fullscreen view
	if m.mode == commentModeClaudePrompt {
		return m.renderClaudePromptView()
	}

	var b strings.Builder

	// Header line
	header := m.headerLine()
	b.WriteString(header + "\n")

	divider := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Render(strings.Repeat("─", m.width))
	b.WriteString(divider + "\n")

	// Content
	if m.isOnThread() {
		b.WriteString(m.renderThread())
	} else {
		b.WriteString(m.renderIssueComment())
	}

	// Reply input area
	if m.mode == commentModeReply {
		b.WriteString("\n" + divider + "\n")
		b.WriteString(m.renderReplyInput())
	}

	// Bottom keybinds
	b.WriteString("\n" + divider + "\n")
	b.WriteString(m.keybindLine())

	return b.String()
}

func (m commentViewModel) emptyView() string {
	var b strings.Builder
	header := lipgloss.NewStyle().
		Foreground(ui.ColorInfo).
		Bold(true).
		Render(fmt.Sprintf("PR #%d · %s", m.prNumber, m.repoAlias))
	b.WriteString(header + "\n\n")
	b.WriteString(ui.StyleDim.Render("  No review comments on this PR.") + "\n\n")
	b.WriteString(ui.StyleDim.Render("  esc:back"))
	return b.String()
}

func (m commentViewModel) headerLine() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.ColorInfo).
		Bold(true)

	title := titleStyle.Render(fmt.Sprintf("PR #%d · %s", m.prNumber, m.repoAlias))

	// Position and status
	pos := fmt.Sprintf("[%d/%d]", m.currentIdx+1, m.totalItems())

	status := ""
	if m.isOnThread() {
		if t := m.currentThread(); t != nil {
			if t.IsResolved {
				status = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render("RESOLVED")
			} else {
				status = lipgloss.NewStyle().Foreground(ui.ColorWarning).Render("UNRESOLVED")
			}
		}
	} else {
		status = ui.StyleDim.Render("GENERAL")
	}

	right := fmt.Sprintf("%s %s", pos, status)
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}

	return title + strings.Repeat(" ", gap) + right
}

func (m commentViewModel) renderThread() string {
	t := m.currentThread()
	if t == nil {
		return ""
	}

	var lines []string

	// File path and line
	pathStyle := lipgloss.NewStyle().Foreground(ui.ColorInfo)
	if t.Line > 0 {
		lines = append(lines, pathStyle.Render(fmt.Sprintf("%s:%d", t.Path, t.Line)))
	} else {
		lines = append(lines, pathStyle.Render(t.Path))
	}

	// Diff hunk (truncated)
	if t.DiffHunk != "" {
		hunkLines := strings.Split(t.DiffHunk, "\n")
		// Show last few lines of the hunk for context
		start := 0
		if len(hunkLines) > 6 {
			start = len(hunkLines) - 6
		}
		addStyle := lipgloss.NewStyle().Foreground(ui.ColorSuccess)
		delStyle := lipgloss.NewStyle().Foreground(ui.ColorDanger)
		hunkStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)

		for _, hl := range hunkLines[start:] {
			display := hl
			if m.width > 4 && len(display) > m.width-4 {
				display = display[:m.width-7] + "..."
			}
			switch {
			case strings.HasPrefix(hl, "+"):
				lines = append(lines, "  "+addStyle.Render(display))
			case strings.HasPrefix(hl, "-"):
				lines = append(lines, "  "+delStyle.Render(display))
			case strings.HasPrefix(hl, "@@"):
				lines = append(lines, "  "+hunkStyle.Render(display))
			default:
				lines = append(lines, "  "+display)
			}
		}
		lines = append(lines, "")
	}

	// Comments
	for _, c := range t.Comments {
		lines = append(lines, m.renderCommentBlock(c.Author, c.Body, c.CreatedAt)...)
		lines = append(lines, "")
	}

	// Apply scroll
	return m.applyScroll(lines)
}

func (m commentViewModel) renderIssueComment() string {
	ic := m.currentIssueComment()
	if ic == nil {
		return ""
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColorInfo).Render("General comment"))
	lines = append(lines, "")
	lines = append(lines, m.renderCommentBlock(ic.Author, ic.Body, ic.CreatedAt)...)

	return m.applyScroll(lines)
}

func (m commentViewModel) renderCommentBlock(author, body string, createdAt time.Time) []string {
	var lines []string

	// Author and time
	authorStyle := lipgloss.NewStyle().Bold(true)
	timeStr := relativeTime(createdAt)
	lines = append(lines, fmt.Sprintf("  %s  %s",
		authorStyle.Render(author),
		ui.StyleDim.Render(timeStr),
	))

	// Separator
	lines = append(lines, "  "+ui.StyleDim.Render("────────────"))

	// Body (word-wrapped)
	bodyWidth := m.width - 6
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	for _, bl := range strings.Split(body, "\n") {
		if len(bl) > bodyWidth {
			// Simple word wrap
			for len(bl) > bodyWidth {
				// Find last space before limit
				cut := bodyWidth
				for cut > 0 && bl[cut] != ' ' {
					cut--
				}
				if cut == 0 {
					cut = bodyWidth
				}
				lines = append(lines, "  "+bl[:cut])
				bl = bl[cut:]
				if len(bl) > 0 && bl[0] == ' ' {
					bl = bl[1:]
				}
			}
			if bl != "" {
				lines = append(lines, "  "+bl)
			}
		} else {
			lines = append(lines, "  "+bl)
		}
	}

	return lines
}

func (m commentViewModel) applyScroll(lines []string) string {
	start := m.scroll
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}

	// Reserve space for header (2), bottom divider+keybinds (2), and reply area if active (4)
	reserved := 4
	if m.mode == commentModeReply {
		reserved += 4
	}
	maxLines := m.height - reserved
	if maxLines < 5 {
		maxLines = 5
	}

	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for _, line := range lines[start:end] {
		b.WriteString(line + "\n")
	}

	if end < len(lines) {
		b.WriteString(ui.StyleDim.Render(
			fmt.Sprintf("  ... %d more lines (j/k to scroll)", len(lines)-end),
		))
	}

	return b.String()
}

func (m commentViewModel) renderReplyInput() string {
	var b strings.Builder

	label := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("Reply:")
	b.WriteString(label + "\n")

	input := string(m.replyBuf)
	if input == "" {
		input = ui.StyleDim.Render("Type your reply... (Enter to send, Esc to cancel)")
	}
	b.WriteString("  " + input + "█\n")

	if m.replyErr != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(ui.ColorDanger).Render("  "+m.replyErr) + "\n")
	}

	return b.String()
}

func (m commentViewModel) renderClaudePromptView() string {
	var b strings.Builder

	divider := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Render(strings.Repeat("─", m.width))

	// Title
	title := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		Render("Launch Claude — PR Review Comment")
	b.WriteString(title + "\n")
	b.WriteString(divider + "\n\n")

	// Show context preview
	contextLabel := lipgloss.NewStyle().Bold(true).Render("Context:")
	b.WriteString(contextLabel + "\n")

	if t := m.currentThread(); t != nil {
		pathStyle := lipgloss.NewStyle().Foreground(ui.ColorInfo)
		if t.Line > 0 {
			b.WriteString("  " + pathStyle.Render(fmt.Sprintf("%s:%d", t.Path, t.Line)) + "\n")
		} else {
			b.WriteString("  " + pathStyle.Render(t.Path) + "\n")
		}

		// Show condensed thread
		for _, c := range t.Comments {
			authorStyle := lipgloss.NewStyle().Bold(true)
			body := c.Body
			if len(body) > 120 {
				body = body[:117] + "..."
			}
			// Replace newlines with spaces for condensed view
			body = strings.ReplaceAll(body, "\n", " ")
			b.WriteString(fmt.Sprintf("  %s: %s\n",
				authorStyle.Render(c.Author),
				ui.StyleDim.Render(body),
			))
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.StyleDim.Render("  Claude will open in plan mode with this context.") + "\n\n")
	b.WriteString(divider + "\n")

	// User input area
	inputLabel := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		Render("Additional instructions (optional):")
	b.WriteString(inputLabel + "\n")

	input := string(m.claudePromptBuf)
	if input == "" {
		b.WriteString("  " + ui.StyleDim.Render("Add details, constraints, or approach preferences...") + "█\n")
	} else {
		b.WriteString("  " + input + "█\n")
	}

	b.WriteString("\n" + divider + "\n")

	// Keybinds
	keyStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(ui.ColorMuted)
	sep := descStyle.Render("  ")

	binds := []string{
		keyStyle.Render("Enter") + descStyle.Render(":launch claude"),
		keyStyle.Render("Esc") + descStyle.Render(":cancel"),
	}
	b.WriteString(strings.Join(binds, sep))

	return b.String()
}

func (m commentViewModel) keybindLine() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(ui.ColorMuted)
	sep := descStyle.Render("  ")

	if m.mode == commentModeReply {
		binds := []string{
			keyStyle.Render("Enter") + descStyle.Render(":send"),
			keyStyle.Render("Esc") + descStyle.Render(":cancel"),
		}
		return strings.Join(binds, sep)
	}

	binds := []string{
		keyStyle.Render("n") + descStyle.Render(":next"),
		keyStyle.Render("p") + descStyle.Render(":prev"),
		keyStyle.Render("R") + descStyle.Render(":reply"),
		keyStyle.Render("C") + descStyle.Render(":claude"),
		keyStyle.Render("o") + descStyle.Render(":open"),
		keyStyle.Render("esc") + descStyle.Render(":back"),
	}
	return strings.Join(binds, sep)
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

// FormatThreadBody formats a review thread as plain text for Claude context.
func FormatThreadBody(thread *github.ReviewThread) string {
	var b strings.Builder
	for _, c := range thread.Comments {
		timeStr := c.CreatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(&b, "%s (%s):\n%s\n\n", c.Author, timeStr, c.Body)
	}
	return b.String()
}
