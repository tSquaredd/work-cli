package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/tui"
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
			result, err := p.Run()
			if err != nil {
				return err
			}

			// If user pressed 'n', drop into the new task wizard
			if m, ok := result.(dashboard.Model); ok && m.NewTaskRequested() {
				return tui.RunNewTask(ws)
			}

			return nil
		},
	}
}
