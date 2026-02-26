package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// RunResume launches the resume task TUI.
func RunResume(ws *workspace.Workspace) error {
	tasks := worktree.CollectTasks(ws)
	if len(tasks) == 0 {
		fmt.Println()
		fmt.Println(ui.StyleDim.Render("  No active tasks to resume."))
		fmt.Println()
		return nil
	}

	// Print header
	subtitle := fmt.Sprintf("%s · %d repos", ws.Root, len(ws.Repos))
	fmt.Println()
	fmt.Println(ui.Header("work · Resume Task", subtitle))
	fmt.Println()

	return runResume(ws, tasks)
}

// resumeTask launches Claude for a specific task by name.
func resumeTask(ws *workspace.Workspace, taskName string) error {
	tasks := worktree.CollectTasks(ws)

	var dirs []string
	var repoNames []string
	for _, t := range tasks {
		if t.Name != taskName {
			continue
		}
		for _, wt := range t.Worktrees {
			dir := worktree.ResolveWorktreeDir(ws, taskName, wt.Alias)
			dirs = append(dirs, dir)
			repoNames = append(repoNames, wt.Alias)
		}
	}

	if len(dirs) == 0 {
		return fmt.Errorf("task %q not found", taskName)
	}

	fmt.Println()
	fmt.Printf("%s %s %s...\n",
		ui.Section("Resuming"),
		ui.StyleInfo.Render(taskName),
		ui.StyleDim.Render(fmt.Sprintf("(%s)", joinComma(repoNames))),
	)
	fmt.Println()

	return claude.Launch(claude.LaunchConfig{
		Workspace: ws,
		TaskName:  taskName,
		Dirs:      dirs,
	})
}

// selectAndResumeTask shows a task picker and resumes the selected one.
func selectAndResumeTask(ws *workspace.Workspace, tasks []worktree.Task) error {
	options := make([]huh.Option[string], len(tasks))
	for i, t := range tasks {
		var repos []string
		for _, wt := range t.Worktrees {
			repos = append(repos, wt.Alias)
		}
		label := fmt.Sprintf("%s  (%s)", t.Name, joinComma(repos))
		options[i] = huh.NewOption(label, t.Name)
	}

	var chosen string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which task?").
				Options(options...).
				Value(&chosen),
		),
	).WithTheme(ui.HuhTheme())

	if err := form.Run(); err != nil {
		return nil
	}

	return resumeTask(ws, chosen)
}
