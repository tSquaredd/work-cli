package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Header renders a styled header box.
func Header(title, subtitle string) string {
	content := StylePrimary.Render(title)
	if subtitle != "" {
		content += "\n" + StyleDim.Render(subtitle)
	}
	return StyleHeader.Render(content)
}

// StatusBadge renders a colored status tag.
func StatusBadge(status string) string {
	switch status {
	case "DIRTY":
		return lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true).
			Render(status)
	case "PUSHED":
		return lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true).
			Render(status)
	case "UNPUSHED":
		return lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true).
			Render(status)
	case "CLEAN":
		return lipgloss.NewStyle().
			Foreground(ColorMuted).
			Render(status)
	default:
		return StyleMuted.Render(status)
	}
}

// TaskCard renders a task with its worktrees in a tree layout.
func TaskCard(taskName string, worktrees []WorktreeInfo) string {
	var b strings.Builder
	b.WriteString("  " + StyleTaskName.Render(taskName) + "\n")

	for i, wt := range worktrees {
		isLast := i == len(worktrees)-1

		// Tree connector
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		// Pad repo name for alignment
		name := StyleRepoName.Render(padRight(wt.Alias, 18))
		branch := StyleBranchName.Render(fmt.Sprintf("(%s)", wt.Branch))
		badge := StatusBadge(wt.Status)

		line := fmt.Sprintf("  %s%s %s  %s",
			StyleTreeBranch.Render(connector),
			name,
			branch,
			badge,
		)
		b.WriteString(line)
		if !isLast {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// WorktreeInfo holds display info for a worktree in a TaskCard.
type WorktreeInfo struct {
	Alias  string
	Branch string
	Status string
}

// ProgressLine renders a result line with a check mark or indicator.
func ProgressLine(repo, message string) string {
	return fmt.Sprintf("  %s %s  %s",
		StyleSuccess.Render("✓"),
		StyleBold.Render(padRight(repo, 18)),
		message,
	)
}

// ErrorLine renders an error result line.
func ErrorLine(repo, message string) string {
	return fmt.Sprintf("  %s %s  %s",
		StyleDanger.Render("✗"),
		StyleBold.Render(padRight(repo, 18)),
		message,
	)
}

// InfoLine renders a dim info line (e.g. linked files).
func InfoLine(repo, message string) string {
	return fmt.Sprintf("    %s  %s",
		StyleDim.Render(padRight(repo, 18)),
		StyleDim.Render(message),
	)
}

// WarningLine renders a yellow warning line.
func WarningLine(repo, message string) string {
	return fmt.Sprintf("  %s %s  %s",
		StyleWarning.Render("!"),
		StyleBold.Render(padRight(repo, 18)),
		message,
	)
}

// Section renders a bold section header.
func Section(title string) string {
	return StyleBold.Render(title)
}

// PRBadge renders a colored PR status icon based on state, review status, and draft state.
func PRBadge(state, reviewStatus string, isDraft bool) string {
	switch state {
	case "MERGED":
		return lipgloss.NewStyle().Foreground(ColorMerged).Render("●")
	case "CLOSED":
		return lipgloss.NewStyle().Foreground(ColorMuted).Render("✗")
	case "OPEN":
		if isDraft {
			return lipgloss.NewStyle().Foreground(ColorMuted).Render("◌")
		}
		switch reviewStatus {
		case "APPROVED":
			return lipgloss.NewStyle().Foreground(ColorSuccess).Render("✓")
		case "CHANGES_REQUESTED":
			return lipgloss.NewStyle().Foreground(ColorWarning).Render("!")
		default:
			return lipgloss.NewStyle().Foreground(ColorInfo).Render("○")
		}
	default:
		return lipgloss.NewStyle().Foreground(ColorInfo).Render("○")
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
