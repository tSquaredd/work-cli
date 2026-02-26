package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

func newCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "clean",
		Short:   "Auto-remove all worktrees with no uncommitted changes",
		Aliases: []string{"prune"},
		RunE: func(cmd *cobra.Command, args []string) error {
			checkForUpdateBg()
			ws := discoverOrDie()
			return runClean(ws)
		},
	}
}

func runClean(ws *workspace.Workspace) error {
	fmt.Println()
	fmt.Println(ui.Section("Auto-cleaning worktrees with no uncommitted changes..."))
	fmt.Println()

	found := false

	// New-style: <workspace>/.worktrees/<task>/<repo>/
	wtBase := filepath.Join(ws.Root, ".worktrees")
	if taskEntries, err := os.ReadDir(wtBase); err == nil {
		for _, taskEntry := range taskEntries {
			if !taskEntry.IsDir() {
				continue
			}
			taskName := taskEntry.Name()
			taskDir := filepath.Join(wtBase, taskName)

			repoEntries, err := os.ReadDir(taskDir)
			if err != nil {
				continue
			}

			for _, repoEntry := range repoEntries {
				if !repoEntry.IsDir() {
					continue
				}
				repoName := repoEntry.Name()
				repo := ws.RepoByAlias(repoName)
				if repo == nil {
					continue
				}
				found = true

				wtDir := filepath.Join(taskDir, repoName)
				branch := worktree.Branch(wtDir)

				if !worktree.IsDirty(wtDir) {
					err := worktree.CleanRemove(repo.Path, wtDir)
					if err != nil {
						fmt.Println(ui.ErrorLine(repoName, fmt.Sprintf("failed to remove %s/%s", repoName, taskName)))
					} else {
						fmt.Println(ui.ProgressLine(repoName, fmt.Sprintf("removed %s/%s (%s)", repoName, taskName, branch)))
					}
				} else {
					fmt.Println(ui.WarningLine(repoName, fmt.Sprintf("kept %s/%s (has changes)", repoName, taskName)))
				}
			}

			// Clean up empty task directory
			_ = os.Remove(taskDir)
		}
	}

	// Old-style: <repo>/.claude/worktrees/<task>/
	for _, repo := range ws.Repos {
		claudeWtDir := filepath.Join(repo.Path, ".claude", "worktrees")
		taskEntries, err := os.ReadDir(claudeWtDir)
		if err != nil {
			continue
		}

		for _, taskEntry := range taskEntries {
			if !taskEntry.IsDir() {
				continue
			}
			found = true
			taskName := taskEntry.Name()
			wtDir := filepath.Join(claudeWtDir, taskName)
			branch := worktree.Branch(wtDir)

			if !worktree.IsDirty(wtDir) {
				err := worktree.CleanRemove(repo.Path, wtDir)
				if err != nil {
					fmt.Println(ui.ErrorLine(repo.Alias, fmt.Sprintf("failed to remove %s/%s", repo.Alias, taskName)))
				} else {
					fmt.Println(ui.ProgressLine(repo.Alias, fmt.Sprintf("removed %s/%s (%s) %s", repo.Alias, taskName, branch, ui.StyleDim.Render("[old location]"))))
				}
			} else {
				fmt.Println(ui.WarningLine(repo.Alias, fmt.Sprintf("kept %s/%s (has changes) %s", repo.Alias, taskName, ui.StyleDim.Render("[old location]"))))
			}
		}

		// Clean up empty .claude/worktrees directory
		_ = os.Remove(claudeWtDir)
	}

	if !found {
		fmt.Println(ui.StyleDim.Render("  No worktrees found."))
	}

	fmt.Println()
	return nil
}
