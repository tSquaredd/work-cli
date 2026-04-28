package commands

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/github"
	"github.com/tSquaredd/work-cli/internal/prstate"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/settings"
	"github.com/tSquaredd/work-cli/internal/tui"
	"github.com/tSquaredd/work-cli/internal/tui/dashboard"
	"github.com/tSquaredd/work-cli/internal/ui"
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
		if latestVersion := checkForUpdateBg(); latestVersion != "" {
			currentVersion := version
			if strings.HasPrefix(currentVersion, "v") {
				currentVersion = currentVersion[1:]
			}
			fmt.Printf("  %s v%s → v%s\n", ui.StyleWarning.Render("Update available:"), currentVersion, latestVersion)
			fmt.Printf("  Update now? (y/n): ")
			var answer string
			fmt.Scanln(&answer)
			fmt.Println()
			if strings.ToLower(strings.TrimSpace(answer)) == "y" {
				return runUpdate()
			}
		}

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
					skip, ok := promptDangerouslySkip()
					if !ok {
						continue
					}
					cfg := claude.LaunchConfig{
						Workspace:                  ws,
						TaskName:                   fmt.Sprintf("review-pr-%d", ctx.PRNumber),
						Dirs:                       []string{ctx.RepoDir},
						ReviewMode:                 true,
						ReviewCtx:                  ctx,
						InitialPrompt:              claude.BuildReviewPrompt(ctx),
						PlanMode:                   isMine,
						DangerouslySkipPermissions: skip,
					}
					_ = claude.SpawnInTab(cfg)
				}
			case m.CommentClaudeRequested():
				ctx := m.CommentClaudeContext()
				taskName := m.CommentTaskName()
				dir := m.CommentWorktreeDir()
				if ctx != nil && taskName != "" && dir != "" {
					skip, ok := promptDangerouslySkip()
					if !ok {
						continue
					}
					cfg := claude.LaunchConfig{
						Workspace:                  ws,
						TaskName:                   taskName,
						Dirs:                       []string{dir},
						Comment:                    ctx,
						InitialPrompt:              claude.BuildCommentPrompt(ctx),
						PlanMode:                   true,
						DangerouslySkipPermissions: skip,
					}
					_ = claude.SpawnInTab(cfg)
				}
			case m.OpenPRRequested():
				taskName := m.SelectedTaskName()
				if taskName != "" {
					_ = tui.RunOpenPR(ws, taskName, m.SelectedWorktreeAlias())
				}
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

// promptDangerouslySkip resolves the --dangerously-skip-permissions decision for
// inline (non-Bubble-Tea) launch sites. It honors the persisted setting:
//   - "always" → returns (true, true) without prompting
//   - "never"  → returns (false, true) without prompting
//   - "ask"    → presents a huh.Confirm. Esc returns (false, false) so callers
//     can abort the launch.
func promptDangerouslySkip() (skip bool, ok bool) {
	s, _ := settings.Load()
	prompt, value := settings.ResolveDangerouslySkip(s)
	if !prompt {
		return value, true
	}

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
		return false, false
	}
	return skip, true
}
