# CLAUDE.md

## What This Is

`work` is a Go CLI tool that manages parallel Claude Code sessions using git worktrees. It auto-discovers git repos in a workspace, creates isolated worktrees per task, and launches Claude with appropriate directory access and deny rules.

Built with the Charm ecosystem: Bubble Tea (TUI), Lip Gloss (styling), Huh (forms), and Cobra (commands).

## Build & Run

```bash
go build -o work ./cmd/work/   # Build binary
go run ./cmd/work/              # Run directly
go vet ./...                    # Static analysis
go test ./...                   # Run tests
```

Version is set via ldflags: `go build -ldflags "-X main.version=2.0.0" -o work ./cmd/work/`

## Project Structure

```
cmd/work/main.go                 # Entry point, version via ldflags
internal/
├── workspace/
│   ├── discovery.go             # find_workspace_root(), Discover()
│   ├── repo.go                  # Repo/Workspace types, alias resolution
│   └── detect.go                # AutoPrefix(), AutoDescription() heuristics
├── worktree/
│   ├── git.go                   # Branch/status inspection (dirty, pushed, etc.)
│   ├── worktree.go              # Create, remove, fetch worktrees
│   ├── link.go                  # Symlink build files (local.properties, .env*)
│   └── task.go                  # CollectTasks() — scan dirs, group by task
├── claude/
│   └── launch.go                # CLAUDE.md gen, settings.local.json, exec claude
├── ui/
│   ├── theme.go                 # Color palette, lipgloss style definitions
│   └── components.go            # Header, StatusBadge, TaskCard, ProgressLine
├── tui/
│   ├── interactive.go           # Root flow (resume/new choice)
│   ├── newtask.go               # Multi-step wizard using huh forms
│   ├── resume.go                # Task selection for resume
│   └── done.go                  # Teardown with confirmations
└── commands/
    ├── root.go                  # Cobra root command, dispatch
    ├── list.go                  # work list (non-interactive, lipgloss output)
    ├── done.go                  # work done (launches TUI)
    ├── clean.go                 # work clean (non-interactive)
    ├── update.go                # work update + background version check
    ├── version.go               # work version
    └── launch.go                # work <repo> <branch> direct launch
```

## Key Design Decisions

- **Git via os/exec**: Shells out to `git` rather than using go-git. Matches bash behavior, simpler, more reliable.
- **Worktree location**: `<workspace>/.worktrees/<task>/<repo>/` (outside repos). Old location `<repo>/.claude/worktrees/<task>/` supported for backward compat.
- **Isolation**: Each Claude session gets deny rules in `settings.local.json` blocking Edit/Write to original repo paths.
- **State is git state**: No database. All task status derived from git (branch existence, push status, dirty state).
- **huh for forms**: Interactive prompts use charmbracelet/huh (Select, MultiSelect, Input, Confirm) instead of raw Bubble Tea models.
- **lipgloss for styled output**: Non-interactive commands (list, clean, version) use lipgloss directly without Bubble Tea.

## Distribution

- **Repo**: `tSquaredd/work-cli` on GitHub
- **Install**: `brew tap tSquaredd/tap && brew install work` or download from GitHub Releases
- **Self-update**: `work update` checks GitHub Releases API
- **Release**: Tag with `vX.Y.Z` → GitHub Actions runs goreleaser → binaries + Homebrew formula
