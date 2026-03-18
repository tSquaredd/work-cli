package dashboard

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/ui"
)

// statusBarModel manages the bottom bar — context-sensitive keybind display.
type statusBarModel struct {
	width          int
	hasTask        bool
	hasActive      bool
	hasPR          bool
	hasComments    bool
	ghAvailable    bool
	showDiff       bool
	showComments   bool
	showDiffView   bool
	showNewTask    bool
	diffViewMode   diffViewMode
	standalonePR   bool // cursor is on a standalone PR row
	isMyPR         bool // cursor is on a "Your PRs" row
	repoHeader     bool // cursor is on a repo header row
	inRepoLevel    bool // cursor is inside a task's worktree list
	message        string // transient status message
	spinner        spinner.Model
	loading        bool
}

func newStatusBarModel() statusBarModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	return statusBarModel{spinner: s}
}

func (m statusBarModel) view() string {
	if m.message != "" {
		return m.messageView()
	}

	return m.keybindView()
}

func (m statusBarModel) keybindView() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.ColorMuted)

	sep := descStyle.Render("  ")

	var binds []string

	// New task overlay
	if m.showNewTask {
		binds = append(binds, keyStyle.Render("Esc")+descStyle.Render(":cancel"))
		return strings.Join(binds, sep)
	}

	// Repo header selected
	if m.repoHeader {
		binds = append(binds, keyStyle.Render("↑↓")+descStyle.Render(":navigate"))
		binds = append(binds, keyStyle.Render("→")+descStyle.Render(":expand"))
		binds = append(binds, keyStyle.Render("←")+descStyle.Render(":collapse"))
		binds = append(binds, keyStyle.Render("n")+descStyle.Render(":new"))
		binds = append(binds, keyStyle.Render("q")+descStyle.Render(":quit"))
		return strings.Join(binds, sep)
	}

	// Standalone PR selected
	if m.standalonePR {
		binds = append(binds, keyStyle.Render("↑↓")+descStyle.Render(":navigate"))
		if m.isMyPR {
			binds = append(binds, keyStyle.Render("r")+descStyle.Render(":resume"))
			binds = append(binds, keyStyle.Render("t")+descStyle.Render(":test"))
		}
		binds = append(binds, keyStyle.Render("d")+descStyle.Render(":diff"))
		if m.ghAvailable {
			binds = append(binds, keyStyle.Render("m")+descStyle.Render(":comments"))
		}
		binds = append(binds, keyStyle.Render("o")+descStyle.Render(":open"))
		binds = append(binds, keyStyle.Render("q")+descStyle.Render(":quit"))
		return strings.Join(binds, sep)
	}

	binds = append(binds, keyStyle.Render("↑↓")+descStyle.Render(":navigate"))

	if m.hasTask {
		binds = append(binds, keyStyle.Render("r")+descStyle.Render(":resume"))

		if m.showDiff {
			binds = append(binds, keyStyle.Render("d")+descStyle.Render(":back"))
		} else {
			binds = append(binds, keyStyle.Render("d")+descStyle.Render(":diff"))
		}

		binds = append(binds, keyStyle.Render("c")+descStyle.Render(":clean"))
		binds = append(binds, keyStyle.Render("t")+descStyle.Render(":test"))

		if m.hasActive {
			binds = append(binds, keyStyle.Render("a")+descStyle.Render(":attach"))
		}

		if m.inRepoLevel {
			binds = append(binds, keyStyle.Render("←")+descStyle.Render(":back"))
		} else {
			binds = append(binds, keyStyle.Render("→")+descStyle.Render(":repos"))
		}

		if m.ghAvailable {
			binds = append(binds, keyStyle.Render("p")+descStyle.Render(":pr"))
		}
		if m.hasPR {
			binds = append(binds, keyStyle.Render("o")+descStyle.Render(":open"))
			binds = append(binds, keyStyle.Render("D")+descStyle.Render(":review"))
		}
		if m.hasComments && m.hasPR {
			binds = append(binds, keyStyle.Render("m")+descStyle.Render(":comments"))
		}
	}

	binds = append(binds, keyStyle.Render("n")+descStyle.Render(":new"))
	binds = append(binds, keyStyle.Render("q")+descStyle.Render(":quit"))

	line := strings.Join(binds, sep)

	// Truncate if too wide
	if m.width > 0 && lipgloss.Width(line) > m.width {
		var short []string
		short = append(short, keyStyle.Render("↑↓")+descStyle.Render(":nav"))
		if m.hasTask {
			short = append(short, keyStyle.Render("r")+descStyle.Render(":resume"))
			short = append(short, keyStyle.Render("d")+descStyle.Render(":diff"))
		}
		short = append(short, keyStyle.Render("n")+descStyle.Render(":new"))
		short = append(short, keyStyle.Render("q")+descStyle.Render(":quit"))
		line = strings.Join(short, sep)
	}

	return line
}

func (m statusBarModel) messageView() string {
	if m.loading {
		return m.spinner.View() + " " + ui.StyleDim.Render(m.message)
	}
	return ui.StyleDim.Render(m.message)
}
