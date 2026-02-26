package commands

import (
	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/tui"
)

func newDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "done",
		Short:   "Pick worktrees to tear down (warns before deleting unpushed work)",
		Aliases: []string{"teardown", "finish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			checkForUpdateBg()
			ws := discoverOrDie()
			return tui.RunDone(ws)
		},
	}
}
