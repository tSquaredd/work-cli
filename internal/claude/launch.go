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
	Workspace    *workspace.Workspace
	TaskName     string
	Dirs         []string         // Worktree directories (all passed via --add-dir)
	Comment      *CommentContext  // optional: PR review comment context
	InitialPrompt string          // optional: initial user message passed via positional arg
	PlanMode     bool             // if true, launch with --permission-mode plan
	ReviewMode   bool             // if true, launch for PR review exploration (no plan mode)
	ReviewCtx    *ReviewContext   // optional: selected diff lines + PR context
}

// CommentContext holds context for launching Claude to address a PR review comment.
type CommentContext struct {
	PRNumber    int
	FilePath    string // e.g. "src/main/Auth.kt"
	Line        int
	DiffHunk    string
	ThreadBody  string // formatted comment thread
	WorktreeDir string // worktree directory for the file
	UserPrompt  string // additional user instructions
}

// ReviewContext holds context for launching Claude from the diff viewer.
type ReviewContext struct {
	PRNumber  int
	PRTitle   string
	RepoAlias string
	RepoDir   string
	FilePath  string
	StartLine int
	EndLine   int
	DiffLines string // the selected diff text
	UserPrompt string
}

// BuildReviewPrompt builds a system prompt for PR review exploration.
func BuildReviewPrompt(ctx *ReviewContext) string {
	lineRange := fmt.Sprintf("line %d", ctx.StartLine)
	if ctx.EndLine > ctx.StartLine {
		lineRange = fmt.Sprintf("lines %d-%d", ctx.StartLine, ctx.EndLine)
	}

	prompt := fmt.Sprintf(`You are helping review PR #%d "%s" in %s.

The user selected code from %s (%s):

%s
The user's question:
%s

You can use gh commands to interact with this PR:
- gh pr diff %d — view full diff
- gh pr view %d — view PR details`,
		ctx.PRNumber, ctx.PRTitle, ctx.RepoAlias,
		ctx.FilePath, lineRange,
		ctx.DiffLines,
		ctx.UserPrompt,
		ctx.PRNumber, ctx.PRNumber)

	return prompt
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

	for _, d := range cfg.Dirs {
		rname := filepath.Base(d)
		branch := worktree.Branch(d)
		desc := "Git repository"
		if repo := cfg.Workspace.RepoByAlias(rname); repo != nil {
			desc = repo.Description
		}

		fmt.Fprintf(&repoTable, "| **%s** | `%s` | `%s` | %s |\n", rname, d, branch, desc)
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

	prompt := fmt.Sprintf(`WORKTREE CONTEXT (task: %s)

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

	if cfg.Comment != nil {
		c := cfg.Comment
		lineInfo := ""
		if c.Line > 0 {
			lineInfo = fmt.Sprintf(" at line %d", c.Line)
		}
		prompt += fmt.Sprintf(`

PR REVIEW COMMENT CONTEXT:
The reviewer left a comment on %s%s:

%s

Comment thread:
%s
Address this review comment. The file is at: %s/%s`,
			c.FilePath, lineInfo, c.DiffHunk, c.ThreadBody, c.WorktreeDir, c.FilePath)
	}

	if cfg.ReviewCtx != nil {
		prompt += "\n\n" + BuildReviewPrompt(cfg.ReviewCtx)
	}

	return prompt
}

// BuildCommentPrompt builds an initial user prompt from a CommentContext.
// The system prompt contains the raw context; this prompt tells Claude what to do.
func BuildCommentPrompt(ctx *CommentContext) string {
	lineInfo := ""
	if ctx.Line > 0 {
		lineInfo = fmt.Sprintf(" at line %d", ctx.Line)
	}

	prompt := fmt.Sprintf("Address this PR review comment on %s%s.\n\nThe file is at: %s/%s",
		ctx.FilePath, lineInfo, ctx.WorktreeDir, ctx.FilePath)

	if ctx.UserPrompt != "" {
		prompt += "\n\nAdditional instructions:\n" + ctx.UserPrompt
	}

	return prompt
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
	for _, d := range cfg.Dirs {
		args = append(args, "--add-dir", d)
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

	// Change to workspace root directory
	if err := os.Chdir(cfg.Workspace.Root); err != nil {
		return fmt.Errorf("changing to workspace root: %w", err)
	}

	return syscall.Exec(claudePath, args, os.Environ())
}

// SpawnInTab launches Claude in a new terminal tab instead of replacing the current process.
// It prepares the session files, opens a tab, and registers the session.
func SpawnInTab(cfg LaunchConfig) error {
	if err := Prepare(cfg); err != nil {
		return err
	}

	// Create tracker early so we can wrap the command with shell PID tracking
	var tracker *session.Tracker
	if cfg.Workspace != nil {
		t, tErr := session.NewTracker(cfg.Workspace.Root)
		if tErr == nil {
			tracker = t
		}
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH — install with: npm install -g @anthropic-ai/claude-code")
	}

	systemPrompt := BuildSystemPrompt(cfg)

	var cmdParts []string
	// Single repo: launch in the repo's worktree directory.
	// Multiple repos: launch in the workspace root.
	launchDir := cfg.Workspace.Root
	if len(cfg.Dirs) == 1 {
		launchDir = cfg.Dirs[0]
	}
	cmdParts = append(cmdParts, fmt.Sprintf("cd %q", launchDir))

	args := []string{claudePath}
	for _, d := range cfg.Dirs {
		args = append(args, "--add-dir", d)
	}
	args = append(args, "--append-system-prompt", fmt.Sprintf("%q", systemPrompt))
	if cfg.PlanMode {
		args = append(args, "--permission-mode", "plan")
	}
	if cfg.InitialPrompt != "" {
		args = append(args, fmt.Sprintf("%q", cfg.InitialPrompt))
	}
	cmdParts = append(cmdParts, strings.Join(args, " "))

	command := strings.Join(cmdParts, " && ")

	// Wrap command with shell PID tracking
	if tracker != nil {
		command = tracker.WrapCommand(cfg.TaskName, command)
	}

	tabTitle := "work: " + cfg.TaskName

	opener := session.DetectTerminal()
	pid, err := opener.OpenTab(command, tabTitle)
	if err != nil {
		return fmt.Errorf("spawning terminal tab: %w", err)
	}

	// Register session
	if tracker != nil {
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
