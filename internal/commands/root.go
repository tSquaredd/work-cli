package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/tui"
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
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle direct launch: work <repo> <branch>
		if len(args) >= 2 {
			return handleDirectLaunch(args)
		}
		if len(args) == 1 {
			return fmt.Errorf("unknown command %q — did you mean: work %s <branch>?", args[0], args[0])
		}

		// Background update check
		checkForUpdateBg()

		ws, err := workspace.Discover()
		if err != nil {
			return err
		}

		return tui.RunInteractive(ws)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(
		newListCmd(),
		newDoneCmd(),
		newCleanCmd(),
		newVersionCmd(),
		newUpdateCmd(),
		newDashboardCmd(),
		newAttachCmd(),
		newPRCmd(),
	)

	// Add aliases as hidden commands
	for _, alias := range []struct {
		name string
		cmd  *cobra.Command
	}{
		{"ls", newListCmd()},
		{"status", newListCmd()},
		{"teardown", newDoneCmd()},
		{"finish", newDoneCmd()},
		{"prune", newCleanCmd()},
		{"dash", newDashboardCmd()},
	} {
		aliasCmd := *alias.cmd
		aliasCmd.Use = alias.name
		aliasCmd.Hidden = true
		rootCmd.AddCommand(&aliasCmd)
	}

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

// discoverOrDie discovers the workspace, printing an error and exiting on failure.
func discoverOrDie() *workspace.Workspace {
	ws, err := workspace.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	return ws
}

// handleDirectLaunch handles `work <repo> <branch>` positional args.
func handleDirectLaunch(args []string) error {
	checkForUpdateBg()

	query := args[0]
	branch := args[1]

	ws, err := workspace.Discover()
	if err != nil {
		return err
	}

	return directLaunch(ws, query, branch)
}

func directLaunch(ws *workspace.Workspace, query, branch string) error {
	repo := ws.ResolveAlias(query)
	if repo == nil {
		return fmt.Errorf("no repo matching %q — available: %s", query, strings.Join(ws.Aliases(), ", "))
	}

	return launchDirect(ws, repo, branch)
}
