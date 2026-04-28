package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/settings"
	"github.com/tSquaredd/work-cli/internal/ui"
)

// settingsModel is the overlay model for the dashboard settings view.
type settingsModel struct {
	form                    *huh.Form
	dangerouslySkipChoice   string
	width, height           int
}

// settingsSavedMsg is emitted when the user completes the settings form. The
// dashboard handles persistence; the overlay only collects choices.
type settingsSavedMsg struct {
	settings settings.Settings
}

// settingsCancelMsg is emitted when the user presses Esc.
type settingsCancelMsg struct{}

func newSettingsModel(s settings.Settings) *settingsModel {
	m := &settingsModel{
		dangerouslySkipChoice: s.DangerouslySkipPermissions,
	}
	if m.dangerouslySkipChoice == "" {
		m.dangerouslySkipChoice = settings.DangerouslySkipAsk
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Skip Claude permission prompts").
				Description("Whether to pass --dangerously-skip-permissions when launching Claude.").
				Options(
					huh.NewOption("Ask each time (default)", settings.DangerouslySkipAsk),
					huh.NewOption("Always skip prompts (--dangerously-skip-permissions)", settings.DangerouslySkipAlways),
					huh.NewOption("Never skip prompts", settings.DangerouslySkipNever),
				).
				Value(&m.dangerouslySkipChoice),
		),
	).WithTheme(ui.HuhTheme()).WithShowHelp(true)

	return m
}

func (m *settingsModel) update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		return func() tea.Msg { return settingsCancelMsg{} }
	}

	if m.form == nil {
		return nil
	}

	model, cmd := m.form.Update(msg)
	m.form = model.(*huh.Form)

	switch m.form.State {
	case huh.StateAborted:
		return func() tea.Msg { return settingsCancelMsg{} }
	case huh.StateCompleted:
		return func() tea.Msg {
			return settingsSavedMsg{
				settings: settings.Settings{
					DangerouslySkipPermissions: m.dangerouslySkipChoice,
				},
			}
		}
	}

	return cmd
}

func (m *settingsModel) view() string {
	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Settings"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, 60))))
	b.WriteString("\n\n")

	if m.form != nil {
		b.WriteString(m.form.View())
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("Saved to %s", settings.Path())))

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2, 4).
		Render(b.String())
}
