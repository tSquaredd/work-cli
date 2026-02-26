package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/tui/dashboard"
)

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "dashboard",
		Short:   "Live dashboard showing all tasks and sessions",
		Aliases: []string{"dash"},
		RunE: func(cmd *cobra.Command, args []string) error {
			checkForUpdateBg()
			ws := discoverOrDie()

			tracker, err := session.NewTracker(ws.Root)
			if err != nil {
				return fmt.Errorf("initializing session tracker: %w", err)
			}

			svc := service.New(ws, tracker)
			model := dashboard.New(svc)

			p := tea.NewProgram(model, tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}
}
