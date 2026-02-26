package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/ui"
)

// statusBarModel manages the bottom bar — context-sensitive keybind display.
type statusBarModel struct {
	width        int
	hasTask      bool
	hasActive    bool
	hasPR        bool
	hasComments  bool
	ghAvailable  bool
	showDiff     bool
	showComments bool
	message      string // transient status message
}

func newStatusBarModel() statusBarModel {
	return statusBarModel{}
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

	binds = append(binds, keyStyle.Render("↑↓")+descStyle.Render(":navigate"))

	if m.hasTask {
		binds = append(binds, keyStyle.Render("r")+descStyle.Render(":resume"))

		if m.showDiff {
			binds = append(binds, keyStyle.Render("d")+descStyle.Render(":back"))
		} else {
			binds = append(binds, keyStyle.Render("d")+descStyle.Render(":diff"))
		}

		binds = append(binds, keyStyle.Render("c")+descStyle.Render(":clean"))

		if m.hasActive {
			binds = append(binds, keyStyle.Render("a")+descStyle.Render(":attach"))
		}

		binds = append(binds, keyStyle.Render("Enter")+descStyle.Render(":expand"))

		if m.ghAvailable {
			binds = append(binds, keyStyle.Render("p")+descStyle.Render(":pr"))
		}
		if m.hasPR {
			binds = append(binds, keyStyle.Render("o")+descStyle.Render(":open"))
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
	return ui.StyleDim.Render(m.message)
}
