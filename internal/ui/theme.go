package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Semantic color palette with adaptive colors for light/dark terminals.
var (
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#D946EF", Dark: "#FF87D7"} // Pink
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#87D787"} // Green
	ColorWarning = lipgloss.AdaptiveColor{Light: "#CA8A04", Dark: "#FFD787"} // Yellow
	ColorDanger  = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#FF8787"} // Red
	ColorInfo    = lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#87D7FF"} // Cyan
	ColorMuted   = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6C6C6C"} // Gray
)

// Base styles.
var (
	StyleBold = lipgloss.NewStyle().Bold(true)
	StyleDim  = lipgloss.NewStyle().Faint(true)

	StylePrimary = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleDanger  = lipgloss.NewStyle().Foreground(ColorDanger)
	StyleInfo    = lipgloss.NewStyle().Foreground(ColorInfo)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
)

// Component styles.
var (
	StyleHeader = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2)

	StyleTaskName = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	StyleTreeBranch = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleRepoName = lipgloss.NewStyle().
			Bold(true)

	StyleBranchName = lipgloss.NewStyle().
			Faint(true)
)

// HuhTheme returns a Huh form theme with a more visible help/hint bar.
func HuhTheme() *huh.Theme {
	t := huh.ThemeBase()

	helpColor := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B8B8B"}
	sepColor := lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6C6C6C"}

	t.Help.ShortKey = t.Help.ShortKey.Foreground(helpColor)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(helpColor)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(sepColor)
	t.Help.FullKey = t.Help.FullKey.Foreground(helpColor)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(helpColor)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(sepColor)
	t.Help.Ellipsis = t.Help.Ellipsis.Foreground(helpColor)

	t.Focused.Title = t.Focused.Title.Foreground(ColorPrimary)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(ColorPrimary).SetString("> ")

	return t
}
