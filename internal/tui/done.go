package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// doneOption represents an item in the done picker.
type doneOption struct {
	Label string
	Type  string // "group" or "single"
	Task  string
	Alias string // empty for group
}

// RunDone launches the teardown TUI.
func RunDone(ws *workspace.Workspace) error {
	tasks := worktree.CollectTasks(ws)
	if len(tasks) == 0 {
		fmt.Println()
		fmt.Println(ui.StyleDim.Render("  No active worktrees to clean up."))
		fmt.Println()
		return nil
	}

	// Print header
	fmt.Println()
	fmt.Println(ui.Header("work · Worktree Cleanup", ""))
	fmt.Println()

	// Build options
	var options []doneOption
	for _, t := range tasks {
		// Group option (only if multiple repos)
		if len(t.Worktrees) > 1 {
			var repos []string
			allSafe := true
			for _, wt := range t.Worktrees {
				repos = append(repos, wt.Alias)
				if wt.Status != worktree.StatusPushed && wt.Status != worktree.StatusClean {
					allSafe = false
				}
			}
			groupTag := "has changes"
			if allSafe {
				groupTag = "all PUSHED"
			}
			label := fmt.Sprintf("%s  (%s)  [%s]",
				t.Name, strings.Join(repos, "+"), groupTag)
			options = append(options, doneOption{
				Label: label,
				Type:  "group",
				Task:  t.Name,
			})
		}

		// Individual options
		for _, wt := range t.Worktrees {
			label := fmt.Sprintf("%s/%s  (%s)  [%s]",
				wt.Alias, t.Name, wt.Branch, wt.Status.String())
			options = append(options, doneOption{
				Label: label,
				Type:  "single",
				Task:  t.Name,
				Alias: wt.Alias,
			})
		}
	}

	// Build huh options
	huhOptions := make([]huh.Option[int], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt.Label, i)
	}

	fmt.Println(ui.StyleDim.Render("  PUSHED = safe  |  UNPUSHED/DIRTY = careful"))
	fmt.Println()

	var selectedIdx int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select a worktree to remove:").
				Options(huhOptions...).
				Value(&selectedIdx),
		),
	).WithTheme(ui.HuhTheme())

	if err := form.Run(); err != nil {
		return nil // User cancelled
	}

	selected := options[selectedIdx]
	fmt.Println()

	// Determine which repos to remove
	type removeItem struct {
		alias string
		task  string
	}
	var toRemove []removeItem

	if selected.Type == "group" {
		for _, t := range tasks {
			if t.Name != selected.Task {
				continue
			}
			for _, wt := range t.Worktrees {
				toRemove = append(toRemove, removeItem{alias: wt.Alias, task: t.Name})
			}
		}
	} else {
		toRemove = append(toRemove, removeItem{alias: selected.Alias, task: selected.Task})
	}

	// Process removals
	for _, item := range toRemove {
		repo := ws.RepoByAlias(item.alias)
		if repo == nil {
			continue
		}
		wtDir := worktree.ResolveWorktreeDir(ws, item.task, item.alias)

		info := worktree.Inspect(wtDir)

		// Confirm dangerous removals
		if info.Status == worktree.StatusDirty {
			fmt.Printf("  %s %s/%s has uncommitted changes!\n",
				ui.StyleDanger.Bold(true).Render("WARNING"),
				item.alias, item.task)

			var confirm bool
			confirmForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Really remove? Changes will be lost.").
						Affirmative("Yes, remove").
						Negative("Skip").
						Value(&confirm),
				),
			).WithTheme(ui.HuhTheme())
			if err := confirmForm.Run(); err != nil || !confirm {
				fmt.Printf("  Skipped %s/%s\n", item.alias, item.task)
				continue
			}
		} else if info.Status == worktree.StatusUnpushed {
			fmt.Printf("  %s %s/%s has unpushed commits on %s\n",
				ui.StyleWarning.Bold(true).Render("NOTE"),
				item.alias, item.task, info.Branch)

			var confirm bool
			confirmForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Remove worktree and delete branch?").
						Affirmative("Yes, remove").
						Negative("Skip").
						Value(&confirm),
				),
			).WithTheme(ui.HuhTheme())
			if err := confirmForm.Run(); err != nil || !confirm {
				fmt.Printf("  Skipped %s/%s\n", item.alias, item.task)
				continue
			}
		}

		// Remove
		err := worktree.Remove(repo.Path, wtDir, true)
		if err != nil {
			fmt.Println(ui.ErrorLine(item.alias, fmt.Sprintf("failed to remove %s/%s", item.alias, item.task)))
		} else {
			fmt.Println(ui.ProgressLine(item.alias, fmt.Sprintf("removed %s/%s (was on %s)", item.alias, item.task, info.Branch)))
		}

		worktree.CleanupTaskDir(ws.Root, item.task)
	}

	fmt.Println()
	return nil
}
