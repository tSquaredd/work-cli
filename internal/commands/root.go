package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/prstate"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/tui"
	"github.com/tSquaredd/work-cli/internal/tui/dashboard"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

var version = "dev"

// SetVersion sets the CLI version (called from main with ldflags value).
func SetVersion(v string) {
	version = v
}

// Version returns the current CLI version.
func Version() string {
	return version
}

var rootCmd = &cobra.Command{
	Use:   "work",
	Short: "Claude Code worktree manager",
	Long:  "Manage parallel Claude Code sessions using git worktrees. Auto-discovers repos in your workspace.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		checkForUpdateBg()

		ws, err := workspace.Discover()
		if err != nil {
			return err
		}

		tracker, err := session.NewTracker(ws.Root)
		if err != nil {
			return fmt.Errorf("initializing session tracker: %w", err)
		}

		svc := service.New(ws, tracker)

		// Initialize PR support
		var prStore *prstate.Store
		var prEnricher *service.PREnricher
		ghAvailable := github.IsAvailable()
		svc.GHAvailable = ghAvailable

		if ghAvailable {
			prStore, err = prstate.NewStore(ws.Root)
			if err == nil {
				svc.PRStore = prStore
				prEnricher = service.NewPREnricher(prStore, ghAvailable)
			}
		}

		for {
			model := dashboard.New(svc)
			if prEnricher != nil {
				model.SetPREnricher(prEnricher)
			}
			p := tea.NewProgram(model, tea.WithAltScreen())
			result, err := p.Run()
			if err != nil {
				return err
			}

			m, ok := result.(dashboard.Model)
			if !ok {
				return nil
			}

			switch {
			case m.DiffViewClaudeRequested():
				ctx := m.DiffViewClaudeContext()
				if ctx != nil {
					isMine := m.DiffViewIsMine()
					cfg := claude.LaunchConfig{
						Workspace:     ws,
						TaskName:      fmt.Sprintf("review-pr-%d", ctx.PRNumber),
						Dirs:          []string{ctx.RepoDir},
						ReviewMode:    true,
						ReviewCtx:     ctx,
						InitialPrompt: claude.BuildReviewPrompt(ctx),
						PlanMode:      isMine,
					}
					_ = claude.SpawnInTab(cfg)
				}
			case m.CommentClaudeRequested():
				ctx := m.CommentClaudeContext()
				taskName := m.CommentTaskName()
				dir := m.CommentWorktreeDir()
				if ctx != nil && taskName != "" && dir != "" {
					cfg := claude.LaunchConfig{
						Workspace:     ws,
						TaskName:      taskName,
						Dirs:          []string{dir},
						Comment:       ctx,
						InitialPrompt: claude.BuildCommentPrompt(ctx),
						PlanMode:      true,
					}
					_ = claude.SpawnInTab(cfg)
				}
			case m.OpenPRRequested():
				taskName := m.SelectedTaskName()
				if taskName != "" {
					_ = tui.RunOpenPR(ws, taskName)
				}
			case m.NewTaskRequested():
				_ = tui.RunNewTaskSpawn(ws)
			default:
				return nil
			}

			// Loop back to dashboard
		}
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(newVersionCmd(), newUpdateCmd())
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return err
	}
	return nil
}
