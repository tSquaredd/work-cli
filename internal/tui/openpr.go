package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/prstate"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// RunOpenPR runs the PR creation wizard for a given task.
func RunOpenPR(ws *workspace.Workspace, taskName string, store *prstate.Store) error {
	// Pre-flight: check gh CLI availability
	if !github.IsAvailable() {
		fmt.Println()
		fmt.Println(ui.StyleDanger.Render("  GitHub CLI (gh) is not installed or not authenticated."))
		fmt.Println(ui.StyleDim.Render("  Install: https://cli.github.com"))
		fmt.Println(ui.StyleDim.Render("  Then run: gh auth login"))
		fmt.Println()
		return fmt.Errorf("gh CLI not available")
	}

	// Build a service to get task details
	svc := service.New(ws, nil)
	task := svc.TaskDetail(taskName)
	if task == nil {
		return fmt.Errorf("task %q not found", taskName)
	}

	fmt.Println()
	fmt.Printf("%s %s\n", ui.Section("Create PR for:"), ui.StyleInfo.Render(taskName))
	fmt.Println()

	// Show worktrees and their push status
	type eligibleWT struct {
		alias  string
		branch string
		dir    string
	}
	var eligible []eligibleWT
	var skipped []string

	for _, wt := range task.Worktrees {
		status := wt.Status

		switch status {
		case worktree.StatusDirty:
			fmt.Println(ui.WarningLine(wt.Alias, "DIRTY — has uncommitted changes, skipping"))
			skipped = append(skipped, wt.Alias)
			continue
		case worktree.StatusClean:
			fmt.Println(ui.InfoLine(wt.Alias, "CLEAN — no changes to push"))
			skipped = append(skipped, wt.Alias)
			continue
		}

		// Check if PR already exists
		if store != nil {
			if rec, ok := store.Get(taskName, wt.Alias); ok && rec.Number > 0 {
				fmt.Println(ui.InfoLine(wt.Alias, fmt.Sprintf("already has PR #%d", rec.Number)))
				continue
			}
		}

		eligible = append(eligible, eligibleWT{
			alias:  wt.Alias,
			branch: wt.Branch,
			dir:    wt.Dir,
		})

		statusStr := "PUSHED"
		if status == worktree.StatusUnpushed {
			statusStr = "UNPUSHED (will push)"
		}
		fmt.Println(ui.ProgressLine(wt.Alias, fmt.Sprintf("%s  branch: %s", statusStr, wt.Branch)))
	}

	if len(eligible) == 0 {
		fmt.Println()
		fmt.Println(ui.StyleDim.Render("  No eligible worktrees for PR creation."))
		return nil
	}

	fmt.Println()

	// Auto-push unpushed worktrees
	for i, wt := range eligible {
		if worktree.HasUnpushed(wt.dir) || !worktree.IsPushed(wt.dir) {
			fmt.Printf("  %s pushing %s...\n", ui.StyleDim.Render("↑"), wt.alias)
			cmd := exec.Command("git", "-C", wt.dir, "push", "-u", "origin", wt.branch)
			if err := cmd.Run(); err != nil {
				fmt.Println(ui.ErrorLine(wt.alias, fmt.Sprintf("push failed: %s", err)))
				// Remove from eligible
				eligible = append(eligible[:i], eligible[i+1:]...)
				continue
			}
			fmt.Println(ui.ProgressLine(wt.alias, "pushed"))
		}
	}

	if len(eligible) == 0 {
		fmt.Println()
		fmt.Println(ui.StyleDim.Render("  No worktrees could be pushed."))
		return nil
	}

	fmt.Println()

	// Target branch selection
	var baseBranch string
	if len(eligible) > 0 {
		recentBranches := worktree.RecentBranches(ws.Repos[0].Path, 15)
		if len(recentBranches) == 0 {
			recentBranches = []string{"main", "develop", "master"}
		}

		branchOptions := make([]huh.Option[string], 0, len(recentBranches))
		for _, b := range recentBranches {
			branchOptions = append(branchOptions, huh.NewOption(b, b))
		}

		baseForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Target branch (base)").
					Description("Which branch should the PR merge into?").
					Options(branchOptions...).
					Value(&baseBranch).
					Height(10),
			),
		).WithTheme(ui.HuhTheme())
		if err := baseForm.Run(); err != nil {
			return nil // user cancelled
		}
	}

	if baseBranch == "" {
		baseBranch = "main"
	}

	// PR title
	defaultTitle := formatTitle(taskName)
	prTitle := defaultTitle
	titleForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("PR title").
				Value(&prTitle),
		),
	).WithTheme(ui.HuhTheme())
	if err := titleForm.Run(); err != nil {
		return nil
	}
	if prTitle == "" {
		prTitle = defaultTitle
	}

	// PR body
	prBody := ""
	bodyForm := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("PR description (optional)").
				Placeholder("Describe what this PR does...").
				Value(&prBody),
		),
	).WithTheme(ui.HuhTheme())
	if err := bodyForm.Run(); err != nil {
		return nil
	}

	// Confirmation
	var confirmed bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Create %d PR(s) → %s?", len(eligible), baseBranch)).
				Value(&confirmed),
		),
	).WithTheme(ui.HuhTheme())
	if err := confirmForm.Run(); err != nil {
		return nil
	}
	if !confirmed {
		fmt.Println(ui.StyleDim.Render("  Cancelled."))
		return nil
	}

	// Create PRs
	fmt.Println()
	fmt.Println(ui.Section("Creating pull requests..."))
	fmt.Println()

	for _, wt := range eligible {
		info, err := github.CreatePR(github.PRCreateParams{
			Dir:   wt.dir,
			Title: prTitle,
			Body:  prBody,
			Base:  baseBranch,
		})

		if err != nil {
			fmt.Println(ui.ErrorLine(wt.alias, fmt.Sprintf("failed: %s", err)))
			continue
		}

		fmt.Println(ui.ProgressLine(wt.alias, fmt.Sprintf("PR #%d created → %s", info.Number, info.URL)))

		// Save to prstate
		if store != nil {
			_ = store.Save(prstate.PRRecord{
				TaskName:  taskName,
				RepoAlias: wt.alias,
				Number:    info.Number,
				URL:       info.URL,
			})
		}
	}

	fmt.Println()
	return nil
}

// formatTitle converts a task name like "auth-refactor" to "Auth refactor".
func formatTitle(taskName string) string {
	s := strings.ReplaceAll(taskName, "-", " ")
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}
