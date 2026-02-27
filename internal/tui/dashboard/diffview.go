package dashboard

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/ui"
)

type lineKind int

const (
	lineContext lineKind = iota
	lineAdded
	lineRemoved
	lineHunkHeader
	lineFileHeader
	lineOther
)

type diffLine struct {
	raw        string
	kind       lineKind
	filePath   string
	newLineNum int // line in new file (for +/context lines)
	oldLineNum int // line in old file (for -/context lines)
}

type diffViewMode int

const (
	diffViewBrowse diffViewMode = iota
	diffViewSelecting
	diffViewCommenting
	diffViewClaudePrompt
)

type claudeReviewContext struct {
	FilePath  string
	StartLine int
	EndLine   int
	DiffText  string
}

type diffViewModel struct {
	// Context
	repoDir   string
	repoAlias string
	prNumber  int
	prTitle   string
	headSHA   string
	isMine    bool

	// Parsed diff
	lines   []diffLine
	fileMap map[int]string // line index -> file path

	// Navigation
	cursor  int // current line in lines[]
	viewTop int // top visible line

	// Visual selection
	mode     diffViewMode
	selStart int

	// Comment composition
	commentBuf []rune

	// Claude prompt
	claudeBuf       []rune
	claudeRequested bool
	claudeContext   *claudeReviewContext

	// Status message
	message string

	// Dimensions
	width, height int
}

func newDiffViewModel() diffViewModel {
	return diffViewModel{
		fileMap: make(map[int]string),
	}
}

func (m *diffViewModel) setData(repoDir, repoAlias string, prNumber int, prTitle, headSHA string, isMine bool, rawDiff string) {
	m.repoDir = repoDir
	m.repoAlias = repoAlias
	m.prNumber = prNumber
	m.prTitle = prTitle
	m.headSHA = headSHA
	m.isMine = isMine
	m.lines = parseDiff(rawDiff)
	m.fileMap = buildFileMap(m.lines)
	m.cursor = 0
	m.viewTop = 0
	m.mode = diffViewBrowse
	m.selStart = 0
	m.commentBuf = nil
	m.claudeBuf = nil
	m.claudeRequested = false
	m.claudeContext = nil
	m.message = ""
}

func (m *diffViewModel) handleKey(key string) (consumed bool) {
	switch m.mode {
	case diffViewCommenting:
		return m.handleCommentKey(key)
	case diffViewClaudePrompt:
		return m.handleClaudeKey(key)
	default:
		return m.handleBrowseKey(key)
	}
}

func (m *diffViewModel) handleBrowseKey(key string) bool {
	switch key {
	case "down":
		m.moveDown()
		return true
	case "up":
		m.moveUp()
		return true
	case "g":
		m.cursor = 0
		m.viewTop = 0
		return true
	case "G":
		if len(m.lines) > 0 {
			m.cursor = len(m.lines) - 1
			m.ensureVisible()
		}
		return true
	case "v":
		if m.mode == diffViewSelecting {
			m.mode = diffViewBrowse
		} else {
			m.mode = diffViewSelecting
			m.selStart = m.cursor
		}
		return true
	case "c":
		m.startComment()
		return true
	case "C":
		m.startClaudePrompt()
		return true
	}
	return false
}

func (m *diffViewModel) handleCommentKey(key string) bool {
	switch key {
	case "esc":
		m.mode = diffViewBrowse
		m.commentBuf = nil
		return true
	case "enter":
		// Signal to post comment — handled in model.go Update
		return true
	case "backspace":
		if len(m.commentBuf) > 0 {
			m.commentBuf = m.commentBuf[:len(m.commentBuf)-1]
		}
		return true
	default:
		if len(key) == 1 {
			m.commentBuf = append(m.commentBuf, rune(key[0]))
		} else if key == "space" {
			m.commentBuf = append(m.commentBuf, ' ')
		}
		return true
	}
}

func (m *diffViewModel) handleClaudeKey(key string) bool {
	switch key {
	case "esc":
		m.mode = diffViewBrowse
		m.claudeBuf = nil
		return true
	case "enter":
		// Signal Claude launch — handled in model.go Update
		return true
	case "backspace":
		if len(m.claudeBuf) > 0 {
			m.claudeBuf = m.claudeBuf[:len(m.claudeBuf)-1]
		}
		return true
	default:
		if len(key) == 1 {
			m.claudeBuf = append(m.claudeBuf, rune(key[0]))
		} else if key == "space" {
			m.claudeBuf = append(m.claudeBuf, ' ')
		}
		return true
	}
}

func (m *diffViewModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.ensureVisible()
	}
}

func (m *diffViewModel) moveDown() {
	if m.cursor < len(m.lines)-1 {
		m.cursor++
		m.ensureVisible()
	}
}

func (m *diffViewModel) ensureVisible() {
	viewportHeight := m.viewportHeight()
	if m.cursor < m.viewTop {
		m.viewTop = m.cursor
	}
	if m.cursor >= m.viewTop+viewportHeight {
		m.viewTop = m.cursor - viewportHeight + 1
	}
}

func (m *diffViewModel) viewportHeight() int {
	h := m.height - 4 // header + dividers + status bar
	if m.mode == diffViewCommenting || m.mode == diffViewClaudePrompt {
		h -= 4 // input area
	}
	if h < 5 {
		h = 5
	}
	return h
}

func (m *diffViewModel) selectionRange() (start, end int) {
	if m.mode == diffViewSelecting {
		s, e := m.selStart, m.cursor
		if s > e {
			s, e = e, s
		}
		return s, e
	}
	return m.cursor, m.cursor
}

func (m *diffViewModel) startComment() {
	m.mode = diffViewCommenting
	m.commentBuf = nil
}

func (m *diffViewModel) startClaudePrompt() {
	m.mode = diffViewClaudePrompt
	m.claudeBuf = nil
}

// commentBody returns the trimmed comment text.
func (m *diffViewModel) commentBody() string {
	return strings.TrimSpace(string(m.commentBuf))
}

// claudeUserPrompt returns the trimmed Claude prompt.
func (m *diffViewModel) claudeUserPrompt() string {
	return strings.TrimSpace(string(m.claudeBuf))
}

// selectedFileAndLines resolves the file path and line range for commenting.
// Returns the file path, startLine, endLine (in the new file), and whether it's valid.
func (m *diffViewModel) selectedFileAndLines() (filePath string, startLine, endLine int, ok bool) {
	selStart, selEnd := m.selectionRange()

	// Find the file path from the selection range
	for i := selStart; i >= 0; i-- {
		if m.lines[i].filePath != "" {
			filePath = m.lines[i].filePath
			break
		}
	}
	if filePath == "" {
		return "", 0, 0, false
	}

	// Find line numbers from the selection — use the new file side
	startLine = 0
	endLine = 0
	for i := selStart; i <= selEnd; i++ {
		if i >= len(m.lines) {
			break
		}
		l := m.lines[i]
		lineNum := l.newLineNum
		if l.kind == lineRemoved {
			lineNum = l.oldLineNum
		}
		if lineNum > 0 {
			if startLine == 0 {
				startLine = lineNum
			}
			endLine = lineNum
		}
	}

	if startLine == 0 {
		return "", 0, 0, false
	}

	return filePath, startLine, endLine, true
}

// selectedDiffText returns the raw diff text in the selection range.
func (m *diffViewModel) selectedDiffText() string {
	selStart, selEnd := m.selectionRange()
	var b strings.Builder
	for i := selStart; i <= selEnd && i < len(m.lines); i++ {
		b.WriteString(m.lines[i].raw)
		b.WriteByte('\n')
	}
	return b.String()
}

func (m diffViewModel) view() string {
	if len(m.lines) == 0 {
		return ui.StyleDim.Render("  No diff content.")
	}

	var b strings.Builder

	// Header
	header := m.headerLine()
	b.WriteString(header + "\n")

	divider := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Render(strings.Repeat("─", m.width))
	b.WriteString(divider + "\n")

	// Diff content
	vpHeight := m.viewportHeight()
	selStart, selEnd := m.selectionRange()

	end := m.viewTop + vpHeight
	if end > len(m.lines) {
		end = len(m.lines)
	}

	addStyle := lipgloss.NewStyle().Foreground(ui.ColorSuccess)
	delStyle := lipgloss.NewStyle().Foreground(ui.ColorDanger)
	hunkStyle := lipgloss.NewStyle().Foreground(ui.ColorInfo)
	fileStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	cursorBg := lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#3A3A3A"})
	selectBg := lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "#DBEAFE", Dark: "#1E3A5F"})
	gutterStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Width(6).Align(lipgloss.Right)

	for i := m.viewTop; i < end; i++ {
		dl := m.lines[i]

		// Gutter
		gutter := ""
		switch dl.kind {
		case lineAdded, lineContext:
			if dl.newLineNum > 0 {
				gutter = fmt.Sprintf("%d", dl.newLineNum)
			}
		case lineRemoved:
			if dl.oldLineNum > 0 {
				gutter = fmt.Sprintf("%d", dl.oldLineNum)
			}
		}
		gutterStr := gutterStyle.Render(gutter) + " "

		// Line content — truncate to fit
		content := dl.raw
		maxWidth := m.width - 8
		if maxWidth > 0 && len(content) > maxWidth {
			content = content[:maxWidth-3] + "..."
		}

		// Apply syntax coloring
		var styled string
		switch dl.kind {
		case lineAdded:
			styled = addStyle.Render(content)
		case lineRemoved:
			styled = delStyle.Render(content)
		case lineHunkHeader:
			styled = hunkStyle.Render(content)
		case lineFileHeader:
			styled = fileStyle.Render(content)
		default:
			styled = content
		}

		line := gutterStr + styled

		// Highlight: cursor or selection
		isCursor := i == m.cursor
		isSelected := m.mode == diffViewSelecting && i >= selStart && i <= selEnd

		if isCursor && isSelected {
			line = selectBg.Bold(true).Render(gutterStr + content)
		} else if isCursor {
			line = cursorBg.Render(gutterStr + content)
		} else if isSelected {
			line = selectBg.Render(gutterStr + content)
		}

		b.WriteString(line + "\n")
	}

	// Input area
	if m.mode == diffViewCommenting {
		b.WriteString(divider + "\n")
		b.WriteString(m.renderCommentInput())
	} else if m.mode == diffViewClaudePrompt {
		b.WriteString(divider + "\n")
		b.WriteString(m.renderClaudeInput())
	}

	// Status message
	if m.message != "" {
		b.WriteString("\n" + ui.StyleDim.Render(m.message))
	}

	// Bottom keybinds
	b.WriteString("\n" + divider + "\n")
	b.WriteString(m.keybindLine())

	return b.String()
}

func (m diffViewModel) headerLine() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.ColorInfo).
		Bold(true)

	title := titleStyle.Render(fmt.Sprintf("PR #%d · %s · %s", m.prNumber, m.repoAlias, m.prTitle))

	pos := fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.lines))
	gap := m.width - lipgloss.Width(title) - len(pos)
	if gap < 2 {
		gap = 2
	}

	return title + strings.Repeat(" ", gap) + pos
}

func (m diffViewModel) renderCommentInput() string {
	var b strings.Builder
	label := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("Comment:")
	b.WriteString(label + "\n")

	input := string(m.commentBuf)
	if input == "" {
		input = ui.StyleDim.Render("Type your comment... (Enter to post, Esc to cancel)")
	}
	b.WriteString("  " + input + "\u2588\n")
	return b.String()
}

func (m diffViewModel) renderClaudeInput() string {
	var b strings.Builder
	label := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("Claude prompt:")
	b.WriteString(label + "\n")

	input := string(m.claudeBuf)
	if input == "" {
		input = ui.StyleDim.Render("Describe what you want Claude to explore... (Enter to launch, Esc to cancel)")
	}
	b.WriteString("  " + input + "\u2588\n")
	return b.String()
}

func (m diffViewModel) keybindLine() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(ui.ColorMuted)
	sep := descStyle.Render("  ")

	var binds []string

	switch m.mode {
	case diffViewCommenting:
		binds = append(binds,
			keyStyle.Render("Enter")+descStyle.Render(":post"),
			keyStyle.Render("Esc")+descStyle.Render(":cancel"),
		)
	case diffViewClaudePrompt:
		binds = append(binds,
			keyStyle.Render("Enter")+descStyle.Render(":launch"),
			keyStyle.Render("Esc")+descStyle.Render(":cancel"),
		)
	case diffViewSelecting:
		binds = append(binds,
			keyStyle.Render("\u2191\u2193")+descStyle.Render(":extend"),
			keyStyle.Render("c")+descStyle.Render(":comment"),
			keyStyle.Render("C")+descStyle.Render(":claude"),
			keyStyle.Render("v")+descStyle.Render(":cancel"),
			keyStyle.Render("esc")+descStyle.Render(":back"),
		)
	default:
		binds = append(binds,
			keyStyle.Render("\u2191\u2193")+descStyle.Render(":navigate"),
			keyStyle.Render("v")+descStyle.Render(":select"),
			keyStyle.Render("c")+descStyle.Render(":comment"),
			keyStyle.Render("C")+descStyle.Render(":claude"),
			keyStyle.Render("o")+descStyle.Render(":open"),
			keyStyle.Render("esc")+descStyle.Render(":back"),
			keyStyle.Render("q")+descStyle.Render(":quit"),
		)
	}

	return strings.Join(binds, sep)
}

// parseDiff parses a unified diff into structured lines with file paths and line numbers.
func parseDiff(raw string) []diffLine {
	rawLines := strings.Split(raw, "\n")
	var result []diffLine

	var currentFile string
	var newLine, oldLine int

	hunkRe := regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

	for _, rl := range rawLines {
		if rl == "" && len(result) > 0 && result[len(result)-1].raw == "" {
			continue // skip consecutive blank lines
		}

		dl := diffLine{raw: rl, filePath: currentFile}

		switch {
		case strings.HasPrefix(rl, "+++ b/"):
			currentFile = rl[6:]
			dl.filePath = currentFile
			dl.kind = lineFileHeader
		case strings.HasPrefix(rl, "--- a/"), strings.HasPrefix(rl, "--- /dev/null"):
			dl.kind = lineFileHeader
		case strings.HasPrefix(rl, "diff --git"):
			dl.kind = lineFileHeader
		case strings.HasPrefix(rl, "index "), strings.HasPrefix(rl, "new file"), strings.HasPrefix(rl, "deleted file"):
			dl.kind = lineOther
		case hunkRe.MatchString(rl):
			dl.kind = lineHunkHeader
			matches := hunkRe.FindStringSubmatch(rl)
			if len(matches) >= 3 {
				oldLine, _ = strconv.Atoi(matches[1])
				newLine, _ = strconv.Atoi(matches[2])
			}
		case strings.HasPrefix(rl, "+"):
			dl.kind = lineAdded
			dl.newLineNum = newLine
			newLine++
		case strings.HasPrefix(rl, "-"):
			dl.kind = lineRemoved
			dl.oldLineNum = oldLine
			oldLine++
		default:
			dl.kind = lineContext
			dl.oldLineNum = oldLine
			dl.newLineNum = newLine
			if rl != "" || len(result) == 0 {
				oldLine++
				newLine++
			}
		}

		result = append(result, dl)
	}

	return result
}

// buildFileMap creates a mapping from line index to file path.
func buildFileMap(lines []diffLine) map[int]string {
	m := make(map[int]string)
	for i, l := range lines {
		if l.filePath != "" {
			m[i] = l.filePath
		}
	}
	return m
}
