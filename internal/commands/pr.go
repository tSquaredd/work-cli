package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/prstate"
	"github.com/tSquaredd/work-cli/internal/tui"
	"github.com/tSquaredd/work-cli/internal/worktree"
)

func newPRCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pr [task-name]",
		Short: "Create pull requests for a task's worktrees",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := discoverOrDie()

			store, err := prstate.NewStore(ws.Root)
			if err != nil {
				return fmt.Errorf("initializing pr store: %w", err)
			}

			var taskName string
			if len(args) > 0 {
				taskName = args[0]
			} else {
				// Auto-detect: if only one task, use it
				tasks := worktree.CollectTasks(ws)
				if len(tasks) == 0 {
					return fmt.Errorf("no active tasks found")
				}
				if len(tasks) == 1 {
					taskName = tasks[0].Name
				} else {
					return fmt.Errorf("multiple tasks found — specify one: work pr <task-name>")
				}
			}

			return tui.RunOpenPR(ws, taskName, store)
		},
	}
}
