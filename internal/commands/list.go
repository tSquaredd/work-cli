package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "Show all active worktrees grouped by task",
		Aliases: []string{"ls", "status"},
		RunE: func(cmd *cobra.Command, args []string) error {
			checkForUpdateBg()
			ws := discoverOrDie()
			return runList(ws)
		},
	}
}

func runList(ws *workspace.Workspace) error {
	tasks := worktree.CollectTasks(ws)

	fmt.Println()
	subtitle := fmt.Sprintf("%s · %d repos", ws.Root, len(ws.Repos))
	fmt.Println(ui.Header("work · Active Worktrees", subtitle))
	fmt.Println()

	if len(tasks) == 0 {
		fmt.Println(ui.StyleDim.Render("  No active worktrees."))
		fmt.Println()
		return nil
	}

	for _, t := range tasks {
		var infos []ui.WorktreeInfo
		for _, wt := range t.Worktrees {
			infos = append(infos, ui.WorktreeInfo{
				Alias:  wt.Alias,
				Branch: wt.Branch,
				Status: wt.Status.String(),
			})
		}
		fmt.Println(ui.TaskCard(t.Name, infos))
		fmt.Println()
	}

	return nil
}
