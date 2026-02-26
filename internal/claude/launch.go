package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/workspace"
	"github.com/tSquaredd/work-cli/internal/worktree"
)

// LaunchConfig holds the parameters for launching Claude.
type LaunchConfig struct {
	Workspace *workspace.Workspace
	TaskName  string
	Dirs      []string // Worktree directories (first is primary CWD)
}

// Prepare generates CLAUDE.md files and settings.local.json deny rules
// for the given worktree directories. Call this before Exec.
func Prepare(cfg LaunchConfig) error {
	if len(cfg.Dirs) == 0 {
		return fmt.Errorf("no worktree directories provided")
	}

	taskDir := filepath.Dir(cfg.Dirs[0])

	// Build repo table and git instructions
	var repoTable strings.Builder
	var gitInstructions strings.Builder
	var repoTablePlain strings.Builder
	var gitRulesPlain strings.Builder

	for i, d := range cfg.Dirs {
		rname := filepath.Base(d)
		branch := worktree.Branch(d)
		desc := "Git repository"
		if repo := cfg.Workspace.RepoByAlias(rname); repo != nil {
			desc = repo.Description
		}

		role := ""
		if i == 0 {
			role = " **(primary CWD)**"
		} else {
			role = " (added via --add-dir)"
		}

		fmt.Fprintf(&repoTable, "| **%s** | `%s` | `%s` | %s%s |\n", rname, d, branch, desc, role)
		fmt.Fprintf(&gitInstructions, "- **%s**: `cd %s` then run git commands\n", rname, d)
		fmt.Fprintf(&repoTablePlain, "  - %s: %s (branch: %s, type: %s)\n", rname, d, branch, desc)
		fmt.Fprintf(&gitRulesPlain, "  - %s: cd %s then run git commands\n", rname, d)
	}

	// Write task-level CLAUDE.md
	taskClaudeMD := fmt.Sprintf(`# Task: %s

You are working in **git worktrees** — isolated checkouts on their own branches, managed by the `+"`work`"+` CLI.

## Your Repositories

| Repo | Worktree Path | Branch | Type |
|------|---------------|--------|------|
%s
## Critical Rules

### Git Commands — ALWAYS use the correct directory
Each repo is a **separate git repository**. You MUST `+"`cd`"+` into the correct worktree directory before running ANY git command (status, diff, commit, log, push, etc.):

%s
### File Edits — Stay in your worktrees
Only modify files inside your worktree directories listed above. Deny rules prevent editing the original repo checkouts (this is enforced automatically).

### Commits — One repo at a time
Each repo is an independent git repository. When committing changes:
- Commit separately in each repo's worktree directory
- Do NOT attempt a single commit spanning multiple repos
- Always `+"`cd`"+` to the correct worktree path before running `+"`git add`"+` / `+"`git commit`"+`

### Plans — Preserve this context
When writing a plan or implementation proposal, you MUST include the full repository table (paths, branches, types) and the git directory rules in the plan itself. Context may be cleared between planning and execution, so the plan must be self-contained with all worktree paths and per-repo instructions.
`, cfg.TaskName, repoTable.String(), gitInstructions.String())

	if err := os.WriteFile(filepath.Join(taskDir, "CLAUDE.md"), []byte(taskClaudeMD), 0o644); err != nil {
		return fmt.Errorf("writing task CLAUDE.md: %w", err)
	}

	// Write per-worktree .claude/CLAUDE.md
	for _, d := range cfg.Dirs {
		claudeDir := filepath.Join(d, ".claude")
		if err := os.MkdirAll(claudeDir, 0o755); err != nil {
			return fmt.Errorf("creating .claude dir: %w", err)
		}

		rname := filepath.Base(d)
		branch := worktree.Branch(d)

		wtClaudeMD := fmt.Sprintf(`# Worktree: %s

You are in repo **%s**, branch `+"`%s`"+`, at `+"`%s`"+`.

## All Repositories in This Task

| Repo | Worktree Path | Branch | Type |
|------|---------------|--------|------|
%s
## Git Commands — Use the correct directory
%s
## Commits — One repo at a time
- Commit separately in each repo's worktree directory
- Do NOT attempt a single commit spanning multiple repos
`, rname, rname, branch, d, repoTable.String(), gitInstructions.String())

		if err := os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte(wtClaudeMD), 0o644); err != nil {
			return fmt.Errorf("writing worktree CLAUDE.md: %w", err)
		}
	}

	// Build deny rules
	denyRules := buildDenyRules(cfg.Workspace)

	// Write settings.local.json to each worktree
	settings := settingsJSON{
		Permissions: permissionsJSON{
			Deny: denyRules,
		},
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	data = append(data, '\n')

	for _, d := range cfg.Dirs {
		claudeDir := filepath.Join(d, ".claude")
		if err := os.MkdirAll(claudeDir, 0o755); err != nil {
			return fmt.Errorf("creating .claude dir: %w", err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), data, 0o644); err != nil {
			return fmt.Errorf("writing settings.local.json: %w", err)
		}
	}

	return nil
}

// BuildSystemPrompt builds the persistent system prompt for the Claude session.
func BuildSystemPrompt(cfg LaunchConfig) string {
	var repoTable strings.Builder
	var gitRules strings.Builder

	for _, d := range cfg.Dirs {
		rname := filepath.Base(d)
		branch := worktree.Branch(d)
		desc := "Git repository"
		if repo := cfg.Workspace.RepoByAlias(rname); repo != nil {
			desc = repo.Description
		}
		fmt.Fprintf(&repoTable, "  - %s: %s (branch: %s, type: %s)\n", rname, d, branch, desc)
		fmt.Fprintf(&gitRules, "  - %s: cd %s then run git commands\n", rname, d)
	}

	return fmt.Sprintf(`WORKTREE CONTEXT (task: %s)

Repositories:
%s
Git rules — ALWAYS cd to the correct worktree directory before ANY git command:
%s
Commit rules:
  - Commit separately in each repo (they are independent git repositories)
  - Do NOT attempt a single commit spanning multiple repos
  - Always cd to the correct worktree path before git add / git commit

Plan preservation:
  - When writing a plan, include the full repo table and git directory rules in the plan itself
  - Context may be compressed between planning and execution — the plan must be self-contained`,
		cfg.TaskName, repoTable.String(), gitRules.String())
}

// Exec replaces the current process with Claude, configured for the given worktrees.
// Before exec, it registers the session PID with the tracker (if workspace root is available).
// The PID survives exec since syscall.Exec replaces the same process.
func Exec(cfg LaunchConfig) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH — install with: npm install -g @anthropic-ai/claude-code")
	}

	systemPrompt := BuildSystemPrompt(cfg)

	args := []string{"claude"}
	for i, d := range cfg.Dirs {
		if i > 0 {
			args = append(args, "--add-dir", d)
		}
	}
	args = append(args, "--append-system-prompt", systemPrompt)

	// Register session PID before exec (PID is preserved across exec)
	if cfg.Workspace != nil {
		tracker, tErr := session.NewTracker(cfg.Workspace.Root)
		if tErr == nil {
			rec := session.SessionRecord{
				TaskName:      cfg.TaskName,
				PID:           os.Getpid(),
				Dirs:          cfg.Dirs,
				LaunchedAt:    time.Now(),
				WorkspaceRoot: cfg.Workspace.Root,
			}
			_ = tracker.Register(rec) // best effort
		}
	}

	// Change to first worktree directory
	if err := os.Chdir(cfg.Dirs[0]); err != nil {
		return fmt.Errorf("changing to worktree directory: %w", err)
	}

	return syscall.Exec(claudePath, args, os.Environ())
}

// SpawnInTab launches Claude in a new terminal tab instead of replacing the current process.
// It prepares the session files, opens a tab, and registers the session.
func SpawnInTab(cfg LaunchConfig) error {
	if err := Prepare(cfg); err != nil {
		return err
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH — install with: npm install -g @anthropic-ai/claude-code")
	}

	systemPrompt := BuildSystemPrompt(cfg)

	var cmdParts []string
	cmdParts = append(cmdParts, fmt.Sprintf("cd %q", cfg.Dirs[0]))

	args := []string{claudePath}
	for i, d := range cfg.Dirs {
		if i > 0 {
			args = append(args, "--add-dir", d)
		}
	}
	args = append(args, "--append-system-prompt", fmt.Sprintf("%q", systemPrompt))
	cmdParts = append(cmdParts, strings.Join(args, " "))

	command := strings.Join(cmdParts, " && ")
	tabTitle := "work: " + cfg.TaskName

	opener := session.DetectTerminal()
	pid, err := opener.OpenTab(command, tabTitle)
	if err != nil {
		return fmt.Errorf("spawning terminal tab: %w", err)
	}

	// Register session
	if cfg.Workspace != nil {
		tracker, tErr := session.NewTracker(cfg.Workspace.Root)
		if tErr == nil {
			rec := session.SessionRecord{
				TaskName:      cfg.TaskName,
				PID:           pid,
				Dirs:          cfg.Dirs,
				LaunchedAt:    time.Now(),
				TerminalTab:   tabTitle,
				WorkspaceRoot: cfg.Workspace.Root,
			}
			_ = tracker.Register(rec)
		}
	}

	return nil
}

// Launch prepares and execs Claude in a single call.
func Launch(cfg LaunchConfig) error {
	if err := Prepare(cfg); err != nil {
		return err
	}
	return Exec(cfg)
}

type settingsJSON struct {
	Permissions permissionsJSON `json:"permissions"`
}

type permissionsJSON struct {
	Deny []string `json:"deny"`
}

func buildDenyRules(ws *workspace.Workspace) []string {
	var rules []string
	for _, repo := range ws.Repos {
		rules = append(rules,
			fmt.Sprintf("Edit(/%s/**)", repo.Path),
			fmt.Sprintf("Write(/%s/**)", repo.Path),
		)
	}
	return rules
}
