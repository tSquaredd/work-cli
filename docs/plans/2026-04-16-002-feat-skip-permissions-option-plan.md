---
title: "feat: Add --dangerously-skip-permissions option to Claude launches"
type: feat
status: active
date: 2026-04-16
origin: docs/brainstorms/skip-permissions-requirements.md
---

# feat: Add --dangerously-skip-permissions option to Claude launches

## Overview

Add an interactive prompt to both the new-task wizard and resume flow that lets users opt into launching Claude Code with `--dangerously-skip-permissions`. The flag is never set programmatically — always gated behind an explicit TUI selection.

## Problem Frame

When launching Claude Code through `work-cli`, Claude runs with default permission behavior — it prompts the user to approve tool calls. Power users who trust their worktree isolation setup want the option to run with `--dangerously-skip-permissions`, but currently there's no way to pass this flag through the CLI. (see origin: `docs/brainstorms/skip-permissions-requirements.md`)

## Requirements Trace

- R1. Add `SkipPermissions bool` to `LaunchConfig`, default `false`
- R2. New `stepSkipPermissions` step in the dashboard new-task wizard between `stepConfigRepo` and `stepCreating`
- R3. Huh overlay prompt on resume before launching Claude
- R4. Append `--dangerously-skip-permissions` to Claude args in `SpawnInTab()`, `resumeTask()`, and `launchNewTask()`
- R5. Mutually exclusive with `--permission-mode plan` in `SpawnInTab()` only
- R6. Esc on the prompt aborts the entire flow
- R7. `SkipPermissions` must only be set from interactive TUI prompts

## Scope Boundaries

- No persisted preference — always prompt fresh
- No CLI flag (e.g., `work --skip-permissions`)
- No change to attach flow (`a` key) — that focuses an existing session
- No changes to `Exec()` or `Launch()` — dead code with no callers
- No changes to `root.go` comment/review `SpawnInTab()` callers — they don't prompt interactively, and `SkipPermissions` defaults to `false`

## Context & Research

### Relevant Code and Patterns

- `internal/claude/launch.go:19-28` — `LaunchConfig` struct; `SpawnInTab()` at line 320 handles `PlanMode` at lines 355-357
- `internal/tui/dashboard/newtask.go:29-35` — step enum (`stepPickRepos`, `stepTaskName`, `stepConfigRepo`, `stepCreating`, `stepDone`); `advanceStep()` at line 350 with three transitions from `stepConfigRepo` to `stepCreating` (lines ~389, ~407, ~443)
- `internal/tui/dashboard/model.go:623-706` — `resumeTask()` inline command builder; `launchNewTask()` at line 1500 reads `m.newTaskView` fields directly
- `internal/tui/dashboard/model.go:42-43` — overlay pattern: `newTaskView *newTaskModel` + `showNewTask bool`; `View()` renders overlay fullscreen at line 1364
- `internal/tui/dashboard/commands.go:222-226` — `newTaskCreatedMsg` carries worktree results; `launchNewTask()` reads skip-permissions from `m.newTaskView` directly (no need to thread through this message)
- Huh form init pattern: `huh.NewForm(huh.NewGroup(...)).WithTheme(ui.HuhTheme()).WithWidth(m.formWidth()).WithShowHelp(true)`

## Key Technical Decisions

- **Overlay for resume (not confirming pattern)**: The `confirming` pattern (y/n in status bar) only supports simple yes/no. The overlay pattern (Huh form rendered fullscreen) matches the requirement for a Select form and is consistent with the new-task wizard aesthetic. (see origin)
- **New step constant (not appended to last config step)**: Adding `stepSkipPermissions` as a discrete step keeps concerns separated and matches the existing step-per-form pattern. (see origin)
- **Skip `Exec()` entirely**: `Exec()` and `Launch()` are dead code with no callers. Adding the flag there would perpetuate dead code.
- **`launchNewTask()` reads from `newTaskModel` directly**: No need to modify `newTaskCreatedMsg` — `launchNewTask()` already accesses `m.newTaskView.taskName` and `m.newTaskView.createdDirs` directly. Same pattern for `skipPermissions`.
- **Resume overlay stores pending task**: The resume skip-permissions overlay needs to remember which task is being resumed. Store a `*service.TaskView` on the overlay model so the resume can proceed after selection.

## Implementation Units

- [ ] **Unit 1: Add SkipPermissions to LaunchConfig and SpawnInTab**

  **Goal:** Establish the data model and wire the flag into the `SpawnInTab()` arg builder, mutually exclusive with `PlanMode`.

  **Requirements:** R1, R4, R5

  **Dependencies:** None

  **Files:**
  - Modify: `internal/claude/launch.go`

  **Approach:**
  - Add `SkipPermissions bool` field to the `LaunchConfig` struct
  - In `SpawnInTab()`, restructure the `PlanMode` block (lines 355-357) into an if/else-if: if `SkipPermissions`, append `--dangerously-skip-permissions`; else if `PlanMode`, append `--permission-mode plan`
  - No changes to `Exec()` — it's dead code

  **Patterns to follow:**
  - Existing `PlanMode` handling in `SpawnInTab()` at lines 355-357
  - `LaunchConfig` field ordering convention (group related booleans)

  **Test scenarios:**
  - Test expectation: none — no test files exist in the project. Verification is manual via the TUI.

  **Verification:**
  - `go build ./...` succeeds
  - Grep for `SkipPermissions` confirms it appears in `LaunchConfig` and `SpawnInTab()`

- [ ] **Unit 2: Add stepSkipPermissions to new-task wizard and thread to launchNewTask**

  **Goal:** Add a skip-permissions Select prompt as a new step in the dashboard new-task wizard, and thread the result into the `launchNewTask()` inline command builder.

  **Requirements:** R2, R4, R6, R7

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `internal/tui/dashboard/newtask.go`
  - Modify: `internal/tui/dashboard/model.go`

  **Approach:**
  - In `newtask.go`:
    - Add `stepSkipPermissions` to the step enum between `stepConfigRepo` and `stepCreating`
    - Add `skipPermissions bool` and `skipPermsChoice string` fields to `newTaskModel`
    - Create `initSkipPermissions()` method following `initTaskName()` pattern — a `huh.NewForm` with a `huh.NewSelect[string]` with Title `"Skip permission prompts?"` and Description `"⚠ Dangerous: Claude will execute tools without asking for approval. Use with caution."`, offering `huh.NewOption("No (default)", "no")` and `huh.NewOption("Yes — skip all permission checks", "yes")`, bound to `m.skipPermsChoice` (initialized to `"no"`), themed with `ui.HuhTheme()`. Use the same Title and Description for the resume overlay prompt.
    - In `advanceStep()`, change the three transitions from `stepConfigRepo → stepCreating` to `stepConfigRepo → stepSkipPermissions`
    - Also in `initConfigRepo()` (line ~194), change the fourth transition where all repos auto-configure in resume-from-PR mode (`m.step = stepCreating`) to `stepSkipPermissions` instead
    - Add a `stepSkipPermissions` case in `advanceStep()`: read `m.skipPermsChoice`, set `m.skipPermissions = (choice == "yes")`, transition to `stepCreating`
    - Add `stepSkipPermissions` rendering to `view()` (if needed — confirm the existing form-rendering path handles it)
  - In `model.go`:
    - In `launchNewTask()` (line ~1536), after building the args slice, add: if `m.newTaskView.skipPermissions`, append `--dangerously-skip-permissions` to args

  **Patterns to follow:**
  - `initTaskName()` in `newtask.go` (lines 139-155) — form creation and binding pattern
  - `stepConfigRepo` case in `advanceStep()` — transition logic
  - `launchNewTask()` reading `m.newTaskView.createdDirs` and `m.newTaskView.taskName` directly

  **Test scenarios:**
  - Test expectation: none — no test files exist in the project. Verification is manual via the TUI.

  **Verification:**
  - `go build ./...` succeeds
  - Create a new task from the dashboard → after branch config, the skip-permissions Select appears
  - Selecting "No (default)" → Claude launches without `--dangerously-skip-permissions`
  - Selecting "Yes" → Claude launches with `--dangerously-skip-permissions`
  - Pressing Esc on the prompt → returns to dashboard, no task created

- [ ] **Unit 3: Add skip-permissions overlay for resume flow and thread to resumeTask**

  **Goal:** Add a Huh overlay prompt before resume launches Claude, and thread the result into the `resumeTask()` inline command builder.

  **Requirements:** R3, R4, R6, R7

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `internal/tui/dashboard/model.go`

  **Approach:**
  - Define a small overlay model struct (e.g., `skipPermsPrompt`) in `model.go` with:
    - `form *huh.Form`
    - `choice string` (bound to the Select)
    - `task *service.TaskView` (the pending task to resume)
    - `init()` method that creates the Huh form (same Select as the new-task wizard)
    - `update(msg)` method that delegates to `m.form.Update(msg)`
    - `view()` method that renders the form
  - Add `resumePrompt *skipPermsPrompt` and `showResumePrompt bool` fields to the dashboard `Model` struct
  - Refactor `resumeTask()` into two parts: (1) `resumeTask()` now only checks for active session (attach flow) then shows the overlay and returns; (2) a new `executeResume(task, skipPermissions)` method contains the existing Prepare + command-building + spawn logic with the flag threaded in. This keeps Prepare() after the user's choice — no wasted work on cancel.
  - Note: `handleResumeFromPR()` at model.go:727 also calls `resumeTask()` for the "matching task exists" case — this path will also show the overlay, which is correct (same prompt on every Claude launch).
  - Define a new message type (e.g., `resumeSkipPermsMsg`) carrying the choice and task reference
  - In `Update()`, handle `resumeSkipPermsMsg`: read the choice, then call `executeResume(task, skipPermissions)`
  - In `Update()`, handle form abort (Esc): set `m.showResumePrompt = false`, return to dashboard
  - In `View()`, when `m.showResumePrompt` is true, render the overlay fullscreen (same pattern as `showNewTask`)
  - In `handleKey()`, when `m.showResumePrompt` is true, delegate to the overlay's update

  **Patterns to follow:**
  - `newTaskView` / `showNewTask` overlay lifecycle in `model.go` (lines 42-43, 259-263, 271-273, 1364-1366)
  - `newTaskFormCancelMsg` handling (lines 237-239)
  - `resumeTask()` existing command-building logic (lines 655-669) — preserve, just add the flag

  **Test scenarios:**
  - Test expectation: none — no test files exist in the project. Verification is manual via the TUI.

  **Verification:**
  - `go build ./...` succeeds
  - Press `r` on an existing task → skip-permissions overlay appears
  - Selecting "No (default)" → Claude resumes without `--dangerously-skip-permissions`
  - Selecting "Yes" → Claude resumes with `--dangerously-skip-permissions`
  - Pressing Esc → returns to dashboard, no Claude session launched
  - Resume on a task with an active session → still focuses existing window (attach flow unchanged)

## System-Wide Impact

- **Interaction graph:** The skip-permissions flag flows through three independent command builders (`SpawnInTab`, `resumeTask` inline, `launchNewTask` inline). Changes to arg construction in one must be mirrored in the others.
- **API surface parity:** The `root.go` callers of `SpawnInTab()` (comment/review launches) are unaffected — they don't set `SkipPermissions` and get the `false` zero-value default.
- **Unchanged invariants:** The attach flow (`a` key / `handleAttach`) is unchanged. Session tracking, worktree creation, and PR workflows are all unaffected.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Step enum iota shift (adding `stepSkipPermissions` changes numeric values of `stepCreating` and `stepDone`) | All comparisons use named constants, not numeric values — safe. Verify with grep. |
| Resume overlay adds complexity to the dashboard model | Keep the overlay model minimal (single form, single message type). Follow the established `newTaskView` pattern exactly. |
| Three independent command builders must stay in sync | Flag handling is a single `if` + `append` in each — hard to get wrong. Verified by manual testing. |

## Sources & References

- **Origin document:** [docs/brainstorms/skip-permissions-requirements.md](docs/brainstorms/skip-permissions-requirements.md)
- Related code: `internal/claude/launch.go` (LaunchConfig, SpawnInTab), `internal/tui/dashboard/newtask.go` (wizard steps), `internal/tui/dashboard/model.go` (resume + launch flows)
