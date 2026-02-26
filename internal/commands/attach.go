package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/session"
)

func newAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <task>",
		Short: "Focus the terminal tab running a task's Claude session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			checkForUpdateBg()
			ws := discoverOrDie()

			tracker, err := session.NewTracker(ws.Root)
			if err != nil {
				return fmt.Errorf("initializing session tracker: %w", err)
			}

			taskName := args[0]
			if err := session.Attach(tracker, taskName); err != nil {
				return err
			}

			fmt.Printf("Focused session for %q\n", taskName)
			return nil
		},
	}
}
