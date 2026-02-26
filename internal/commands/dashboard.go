package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/prstate"
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

			// Initialize PR support
			var prStore *prstate.Store
			var prEnricher *service.PREnricher
			ghAvailable := github.IsAvailable()
			svc.GHAvailable = ghAvailable

			if ghAvailable {
				prStore, err = prstate.NewStore(ws.Root)
				if err == nil {
					svc.PRStore = prStore
					prEnricher = service.NewPREnricher(prStore, ghAvailable)
				}
			}

			for {
				model := dashboard.New(svc)
				if prEnricher != nil {
					model.SetPREnricher(prEnricher)
				}
				p := tea.NewProgram(model, tea.WithAltScreen())
				result, err := p.Run()
				if err != nil {
					return err
				}

				m, ok := result.(dashboard.Model)
				if !ok {
					return nil
				}

				switch {
				case m.OpenPRRequested():
					taskName := m.SelectedTaskName()
					if taskName != "" && prStore != nil {
						_ = tui.RunOpenPR(ws, taskName, prStore)
					}
				case m.NewTaskRequested():
					// Run new task wizard outside alt-screen, spawn Claude in new window
					_ = tui.RunNewTaskSpawn(ws)
				default:
					return nil
				}

				// Loop back to dashboard
			}
		},
	}
}
