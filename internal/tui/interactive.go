package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// RunInteractive launches the main interactive TUI.
func RunInteractive(ws *workspace.Workspace) error {
	tasks := worktree.CollectTasks(ws)

	// Print header
	subtitle := fmt.Sprintf("%s · %d repos", ws.Root, len(ws.Repos))
	fmt.Println()
	fmt.Println(ui.Header("work · Claude Worktree Manager", subtitle))
	fmt.Println()

	if len(tasks) > 0 {
		// Show in-flight tasks
		fmt.Println(ui.Section("In flight:"))
		fmt.Println()
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

		// Choose: resume or new task
		var action string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("What would you like to do?").
					Options(
						huh.NewOption("Resume an existing task", "resume"),
						huh.NewOption("Start a new task", "new"),
					).
					Value(&action),
			),
		).WithTheme(ui.HuhTheme())

		if err := form.Run(); err != nil {
			return nil // User cancelled
		}

		if action == "resume" {
			return runResume(ws, tasks)
		}
	}

	return RunNewTask(ws)
}

// runResume handles resuming an existing task.
func runResume(ws *workspace.Workspace, tasks []worktree.Task) error {
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks to resume")
	}

	// Build task options
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
		return nil // User cancelled
	}

	// Find the chosen task
	var dirs []string
	var repoNames []string
	for _, t := range tasks {
		if t.Name != chosen {
			continue
		}
		for _, wt := range t.Worktrees {
			dir := worktree.ResolveWorktreeDir(ws, chosen, wt.Alias)
			dirs = append(dirs, dir)
			repoNames = append(repoNames, wt.Alias)
		}
	}

	fmt.Println()
	fmt.Printf("%s %s %s...\n",
		ui.Section("Resuming"),
		ui.StyleInfo.Render(chosen),
		ui.StyleDim.Render(fmt.Sprintf("(%s)", joinComma(repoNames))),
	)
	fmt.Println()

	return claude.Launch(claude.LaunchConfig{
		Workspace: ws,
		TaskName:  chosen,
		Dirs:      dirs,
	})
}

func joinComma(items []string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += ", "
		}
		result += item
	}
	return result
}

// RunInteractiveProgram runs the interactive TUI as a Bubble Tea program.
// This is an alternative to RunInteractive for when we need full TUI control.
func RunInteractiveProgram(ws *workspace.Workspace) error {
	p := tea.NewProgram(newInteractiveModel(ws))
	_, err := p.Run()
	return err
}

type interactiveModel struct {
	ws    *workspace.Workspace
	tasks []worktree.Task
	done  bool
}

func newInteractiveModel(ws *workspace.Workspace) interactiveModel {
	return interactiveModel{
		ws:    ws,
		tasks: worktree.CollectTasks(ws),
	}
}

func (m interactiveModel) Init() tea.Cmd {
	return nil
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m interactiveModel) View() string {
	if m.done {
		return ""
	}
	return "Interactive mode — use RunInteractive() for the huh-based flow"
}
