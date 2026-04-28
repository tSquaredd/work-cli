package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/settings"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// repoConfig holds per-repo branch configuration.
type repoConfig struct {
	Alias      string
	Branch     string
	BaseBranch string
}

// RunNewTask runs the new task wizard and exec's Claude (replaces process).
func RunNewTask(ws *workspace.Workspace) error {
	return newTaskWizard(ws, claude.Launch)
}

// RunNewTaskSpawn runs the new task wizard and spawns Claude in a new terminal
// window, returning control to the caller.
func RunNewTaskSpawn(ws *workspace.Workspace) error {
	return newTaskWizard(ws, claude.SpawnInTab)
}

// newTaskWizard is the shared multi-step new task wizard.
// The launch function determines how Claude is started (exec vs spawn in tab).
func newTaskWizard(ws *workspace.Workspace, launch func(claude.LaunchConfig) error) error {
	// Step 1: Pick repo(s)
	repos, err := pickRepos(ws)
	if err != nil {
		return nil // User cancelled
	}
	if len(repos) == 0 {
		fmt.Println("No repos selected. Exiting.")
		return nil
	}

	fmt.Println()
	var repoNames []string
	for _, r := range repos {
		repoNames = append(repoNames, r.Alias)
	}
	fmt.Printf("Selected: %s\n", ui.StyleInfo.Render(strings.Join(repoNames, ", ")))
	fmt.Println()

	// Step 2: Task name
	taskName, err := inputTaskName()
	if err != nil || taskName == "" {
		fmt.Println("No task name. Exiting.")
		return nil
	}

	// Step 3: Branch config per repo
	fmt.Println()
	fmt.Println(ui.Section("Configure each repo:"))
	fmt.Println()

	configs := make([]repoConfig, len(repos))
	for i, repo := range repos {
		cfg, err := configureRepo(repo, taskName)
		if err != nil {
			return nil // User cancelled
		}
		configs[i] = cfg
		fmt.Println()
	}

	// Step 4: Create worktrees
	fmt.Printf("%s %s\n", ui.Section("Creating worktrees for:"), ui.StyleInfo.Render(taskName))
	fmt.Println()

	var dirs []string
	for _, cfg := range configs {
		repo := ws.RepoByAlias(cfg.Alias)
		if repo == nil {
			continue
		}
		wtDir := worktree.WorktreeDir(ws, taskName, cfg.Alias)

		// Fetch
		fmt.Printf("  %s fetching origin/%s...\n", ui.StyleDim.Render(cfg.Alias), cfg.BaseBranch)
		_ = worktree.Fetch(repo.Path, cfg.BaseBranch)

		// Create
		result := worktree.Create(worktree.CreateConfig{
			RepoDir:     repo.Path,
			WorktreeDir: wtDir,
			Branch:      cfg.Branch,
			BaseBranch:  cfg.BaseBranch,
		})

		if result.Error != nil {
			fmt.Println(ui.ErrorLine(cfg.Alias, fmt.Sprintf("failed: %s", result.Error)))
			continue
		}

		if !result.Created {
			fmt.Println(ui.WarningLine(cfg.Alias, fmt.Sprintf("worktree '%s' already exists", taskName)))
		} else if result.Attached {
			fmt.Println(ui.ProgressLine(cfg.Alias, fmt.Sprintf("attached to existing branch %s", cfg.Branch)))
		} else {
			fmt.Println(ui.ProgressLine(cfg.Alias, fmt.Sprintf("created worktree (%s from origin/%s)", cfg.Branch, cfg.BaseBranch)))
		}

		// Link build files
		linkResult := worktree.LinkBuildFiles(result.Dir, repo.Path)
		if len(linkResult.Files) > 0 {
			fmt.Println(ui.InfoLine(cfg.Alias, fmt.Sprintf("linked %s", strings.Join(linkResult.Files, ", "))))
		}

		dirs = append(dirs, result.Dir)
	}

	if len(dirs) == 0 {
		fmt.Println()
		fmt.Println(ui.StyleDanger.Render("  No worktrees were created. Nothing to launch."))
		return fmt.Errorf("no worktrees created")
	}

	// Resolve --dangerously-skip-permissions per user settings.
	skipPerms, err := resolveDangerouslySkip()
	if err != nil {
		return nil // user cancelled
	}

	fmt.Println()
	fmt.Println(ui.Section("Launching Claude..."))
	fmt.Println()

	return launch(claude.LaunchConfig{
		Workspace:                  ws,
		TaskName:                   taskName,
		Dirs:                       dirs,
		DangerouslySkipPermissions: skipPerms,
	})
}

// resolveDangerouslySkip checks the persisted setting and prompts the user when
// it is "ask". Returns the boolean to pass into LaunchConfig. An error from this
// function indicates the user cancelled the prompt.
func resolveDangerouslySkip() (bool, error) {
	s, _ := settings.Load()
	prompt, value := settings.ResolveDangerouslySkip(s)
	if !prompt {
		return value, nil
	}

	var skip bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(settings.DangerouslySkipPromptTitle).
				Description(settings.DangerouslySkipPromptDescription).
				Affirmative("Yes").
				Negative("No").
				Value(&skip),
		),
	).WithTheme(ui.HuhTheme())
	if err := form.Run(); err != nil {
		return false, err
	}
	return skip, nil
}

func pickRepos(ws *workspace.Workspace) ([]workspace.Repo, error) {
	options := make([]huh.Option[string], len(ws.Repos))
	for i, r := range ws.Repos {
		label := fmt.Sprintf("%s  — %s", r.Alias, r.Description)
		options[i] = huh.NewOption(label, r.Alias)
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which repo(s) are you working in?").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(ui.HuhTheme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	var repos []workspace.Repo
	for _, alias := range selected {
		if r := ws.RepoByAlias(alias); r != nil {
			repos = append(repos, *r)
		}
	}
	return repos, nil
}

func inputTaskName() (string, error) {
	var name string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Task name").
				Placeholder("e.g. auth-refactor, remove-parse-session").
				Value(&name),
		),
	).WithTheme(ui.HuhTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return sanitizeTaskName(name), nil
}

func sanitizeTaskName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	return re.ReplaceAllString(name, "")
}

func configureRepo(repo workspace.Repo, taskName string) (repoConfig, error) {
	defaultBranch := repo.Prefix + "-" + taskName

	// Fetch remote refs so AllBranches can include remote-only branches.
	worktree.FetchAll(repo.Path)

	// Get all branches for base selection
	allBranches := worktree.AllBranches(repo.Path)
	currentBranch := worktree.CurrentBranch(repo.Path)

	var branch string
	var baseBranch string

	// Branch name input
	branch = defaultBranch
	branchForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("%s — Branch name", repo.Alias)).
				Value(&branch),
		),
	).WithTheme(ui.HuhTheme())
	if err := branchForm.Run(); err != nil {
		return repoConfig{}, err
	}
	if branch == "" {
		branch = defaultBranch
	}

	// Base branch selection
	if len(allBranches) > 0 {
		branchOptions := make([]huh.Option[string], 0, len(allBranches))
		for _, b := range allBranches {
			label := b
			if b == currentBranch {
				label = b + " (current)"
			}
			branchOptions = append(branchOptions, huh.NewOption(label, b))
		}

		baseForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("%s — Base branch", repo.Alias)).
					Description("Fetches latest before creating worktree").
					Options(branchOptions...).
					Value(&baseBranch).
					Filtering(true).
					Height(10),
			),
		).WithTheme(ui.HuhTheme())
		if err := baseForm.Run(); err != nil {
			baseBranch = currentBranch
		}
	}

	if baseBranch == "" {
		baseBranch = currentBranch
	}

	fmt.Printf("  %s  branch: %s  base: %s\n",
		ui.StyleBold.Render(repo.Alias),
		ui.StyleInfo.Render(branch),
		ui.StyleDim.Render(baseBranch),
	)

	return repoConfig{
		Alias:      repo.Alias,
		Branch:     branch,
		BaseBranch: baseBranch,
	}, nil
}
