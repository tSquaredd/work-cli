# Skip Permissions Option

**Date:** 2026-04-16
**Status:** Ready for planning
**Scope:** Lightweight

## Problem

When launching Claude Code through `work-cli`, Claude runs with default permission behavior — it prompts the user to approve tool calls. Power users who trust their worktree isolation setup want the option to run with `--dangerously-skip-permissions`, but currently there's no way to pass this flag through the CLI.

## Goal

Users can opt into `--dangerously-skip-permissions` each time they launch Claude, both when creating a new task and when resuming an existing one.

## Requirements

1. **New field on `LaunchConfig`**: Add `SkipPermissions bool` to `internal/claude/launch.go:LaunchConfig`. Default `false`.

2. **Prompt in new-task wizard**: Add a new `stepSkipPermissions` step between `stepConfigRepo` and `stepCreating` in the dashboard overlay wizard (`internal/tui/dashboard/newtask.go`). Show a Huh Select with "No (default)" and "Yes" options. Store the result on `newTaskModel` and thread it through `newTaskCreatedMsg` to `launchNewTask()`. Note: `internal/tui/newtask.go` is dead code with no callers — the live flow is the dashboard overlay.

3. **Prompt on resume**: When resuming a task via the dashboard (`r` key, `internal/tui/dashboard/model.go:resumeTask`), show a Huh overlay prompt before launching Claude. Use the same overlay pattern as the new-task wizard (a small Huh form model rendered over the dashboard). The resume path builds the Claude command inline rather than calling `SpawnInTab`, so the flag needs to be threaded through that inline builder.

4. **Flag passed to Claude**: When `SkipPermissions` is true, append `--dangerously-skip-permissions` to the Claude args in `SpawnInTab()` in `internal/claude/launch.go`, and in the inline command builders in `model.go:resumeTask` and `model.go:launchNewTask` (both build Claude commands inline rather than calling `SpawnInTab`).

5. **Mutually exclusive with plan mode**: In `SpawnInTab()` only (the only place `--permission-mode plan` is passed), when `SkipPermissions` is true, do not also pass `--permission-mode plan`. The other command builders (`resumeTask`, `launchNewTask`) never pass plan mode, so no guard is needed there.

6. **Cancel aborts the flow**: Pressing Esc on the skip-permissions prompt aborts the entire new-task or resume operation (matching how Esc works on every other wizard step). The user must explicitly select "No" or "Yes" to proceed.

7. **Interactive prompt only**: `SkipPermissions` must only be set from interactive TUI prompts, never from environment variables, config files, or programmatic callers.

## Non-goals

- Persisting the user's preference across sessions (always prompt fresh)
- Adding a CLI flag to bypass the prompt (e.g., `work --skip-permissions`)
- Changing behavior of the attach flow (`a` key) — that focuses an existing session, not launching a new one

## Touch Points

| File | Change |
|------|--------|
| `internal/claude/launch.go` | Add `SkipPermissions` to `LaunchConfig`; thread into `SpawnInTab()` args |
| `internal/tui/dashboard/newtask.go` | Add skip-permissions prompt step in the dashboard new-task overlay wizard; pass value through to launch |
| `internal/tui/dashboard/model.go` | Add skip-permissions prompt in `resumeTask()` before building Claude command; thread into args in both `resumeTask()` and `launchNewTask()` inline builders |
