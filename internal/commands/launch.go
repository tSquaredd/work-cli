package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// launchDirect handles `work <repo> <branch>` — creates worktree and launches Claude.
func launchDirect(ws *workspace.Workspace, repo *workspace.Repo, branch string) error {
	taskName := sanitizeName(branch)
	wtDir := worktree.WorktreeDir(ws, taskName, repo.Alias)

	result := worktree.CreateDirect(repo.Path, wtDir, branch)
	if result.Error != nil {
		return fmt.Errorf("creating worktree: %w", result.Error)
	}

	if result.Created {
		fmt.Println(ui.ProgressLine(repo.Alias, fmt.Sprintf("created worktree (branch: %s)", branch)))
	}

	// Link build files
	linkResult := worktree.LinkBuildFiles(result.Dir, repo.Path)
	if len(linkResult.Files) > 0 {
		fmt.Println(ui.InfoLine(repo.Alias, fmt.Sprintf("linked %s", strings.Join(linkResult.Files, ", "))))
	}

	return claude.Launch(claude.LaunchConfig{
		Workspace: ws,
		TaskName:  taskName,
		Dirs:      []string{result.Dir},
	})
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9-]`)
	return re.ReplaceAllString(name, "")
}
