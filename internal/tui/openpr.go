package tui

import (
	"fmt"
	"os/exec"

	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// RunOpenPR pushes unpushed worktrees and opens the browser for PR creation.
func RunOpenPR(ws *workspace.Workspace, taskName string) error {
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

	// Open browser for each eligible worktree
	for _, wt := range eligible {
		fmt.Printf("  %s opening browser for %s...\n", ui.StyleDim.Render("→"), wt.alias)
		if err := github.CreateInBrowser(wt.dir); err != nil {
			fmt.Println(ui.ErrorLine(wt.alias, fmt.Sprintf("failed to open browser: %s", err)))
			continue
		}
		fmt.Println(ui.ProgressLine(wt.alias, "opened in browser"))
	}

	fmt.Println()
	return nil
}
