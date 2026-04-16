---
title: "feat: Add Warp terminal support"
type: feat
status: active
date: 2026-04-16
origin: docs/brainstorms/warp-terminal-support-requirements.md
---

# feat: Add Warp terminal support

## Overview

Add a `warpOpener` implementation of the `TabOpener` interface so that Warp terminal users get the same tab-launch and tab-focus experience currently available to Ghostty, iTerm2, and Terminal.app users. The implementation mirrors `ghosttyOpener`'s AppleScript UI scripting approach since Warp lacks a native AppleScript dictionary.

## Problem Frame

Users running Warp terminal (`TERM_PROGRAM=WarpTerminal`) fall through to the `fallbackOpener`, which prints the command for manual copy-paste execution. This breaks the core UX of launching and resuming Claude Code sessions directly from the dashboard. (see origin: `docs/brainstorms/warp-terminal-support-requirements.md`)

## Requirements Trace

- R1. **Detection**: `DetectTerminal()` returns a `warpOpener` when `TERM_PROGRAM=WarpTerminal`
- R2. **OpenTab**: Opens a new Warp tab and executes the provided command via AppleScript UI scripting. Sets tab title via ANSI escape.
- R3. **FocusTab**: Activates Warp and cycles through tabs to find one matching the identifier.
- R4. **Command file pattern**: Uses existing `writeCommandFile()` helper to avoid AppleScript escaping issues.

## Scope Boundaries

- No Warp CLI cold-start fallback (no `-e` flag equivalent exists)
- No Warp URI scheme integration (`warp://action/new_tab` cannot execute commands)
- No Warp Tab Configs integration (cannot be triggered programmatically)
- No new test files (matching existing project convention)

## Context & Research

### Relevant Code and Patterns

- `internal/session/terminal.go` — all terminal handlers live here, `TabOpener` interface defined at line 35
- `ghosttyOpener` (lines 57-166) — direct template for the Warp implementation; uses System Events UI scripting for both `OpenTab` and `FocusTab`
- `writeCommandFile()` (lines 13-32) — shared helper that writes commands to temp scripts to avoid AppleScript escaping issues
- `DetectTerminal()` (lines 42-54) — switch on `strings.ToLower(os.Getenv("TERM_PROGRAM"))`

### Institutional Learnings

- No `docs/solutions/` directory exists in this project

## Key Technical Decisions

- **AppleScript-only (no CLI fallback)**: Ghostty has a CLI fallback via `ghostty -e`. Warp has no equivalent. If Warp is not running, `OpenTab` returns an error like the other handlers. This is acceptable because the tool is run from within a terminal — Warp will always be running when `TERM_PROGRAM=WarpTerminal` is set.
- **Mirror Ghostty's FocusTab approach**: Cycle through tabs using Cmd+Shift+] and check window title. Same limitation as Ghostty (visible cycling, max 20 tabs).
- **Process name in System Events**: Use `"Warp"` — this is the process name macOS reports for the Warp application.
- **ANSI title escape**: Use `printf '\033]0;title\007'` prefix in the command, same as Ghostty, to set the tab title for later `FocusTab` identification.

## Implementation Units

- [ ] **Unit 1: Add warpOpener struct and wire into DetectTerminal**

  **Goal:** Add a complete `warpOpener` implementing `TabOpener` and make it discoverable via `DetectTerminal()`.

  **Requirements:** R1, R2, R3, R4

  **Dependencies:** None

  **Files:**
  - Modify: `internal/session/terminal.go`

  **Approach:**
  - Add `case "warpterminal":` to the `DetectTerminal()` switch (the env var value `WarpTerminal` becomes `warpterminal` after `strings.ToLower()`)
  - Add `warpOpener` struct with `OpenTab` and `FocusTab` methods
  - `OpenTab`: Follows `ghosttyOpener.openViaAppleScript` pattern — call `writeCommandFile()`, build AppleScript that activates Warp via System Events, sends Cmd+T for new tab, types the command, presses Return. Prepend ANSI title escape to command like Ghostty does.
  - `FocusTab`: Follows `ghosttyOpener.FocusTab` pattern — activate Warp, cycle tabs with Cmd+Shift+] checking window title for the identifier.
  - No CLI fallback method needed (unlike Ghostty's two-pronged approach).
  - Place the new code between `ghosttyOpener` and `iterm2Opener` sections for logical grouping (UI-scripted terminals together).

  **Patterns to follow:**
  - `ghosttyOpener.openViaAppleScript()` for `OpenTab` structure
  - `ghosttyOpener.FocusTab()` for tab cycling approach
  - Error wrapping with `fmt.Errorf("context: %w", err)` per CLAUDE.md conventions
  - Cleanup function call on AppleScript failure, same as all other handlers

  **Test scenarios:**
  - Test expectation: none — matching existing project convention (no test files exist)

  **Verification:**
  - `go vet ./...` passes
  - `go build ./cmd/work/` succeeds
  - Manual test in Warp: creating a new task from the dashboard opens a new Warp tab and executes Claude in it
  - Manual test in Warp: attaching to an existing session focuses the correct tab
  - Running in a non-Warp terminal still works as before (no regression)

## System-Wide Impact

- **Interaction graph:** No new callbacks or middleware. The change adds a new case to an existing switch statement and a new struct implementing an existing interface.
- **Error propagation:** Follows the same pattern as all existing handlers — errors from AppleScript execution are wrapped and returned to callers in `claude/launch.go` and `tui/dashboard/model.go`.
- **API surface parity:** All four call sites (`SpawnInTab` in `launch.go`, `resumeTask` and `launchNewTask` in `model.go`, `Attach` in `attach.go`) use `DetectTerminal()` and automatically pick up the new handler.
- **Unchanged invariants:** Session tracking via PID files is terminal-agnostic and unaffected. The `TabOpener` interface is unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Warp's System Events process name may differ from `"Warp"` on some installs | Manual testing will verify; the name is standard for macOS app bundles |
| Warp may handle Cmd+Shift+] differently for tab cycling | Using the same key code (30) as Ghostty; standard macOS tab navigation shortcut |
| Warp's input block model may interfere with keystroke injection | Warp processes input differently from traditional terminals (blocks, AI suggestions). Manual testing should verify that System Events keystrokes are received by the shell prompt and that Warp features like AI command suggestions do not intercept the typed command |
| ANSI title escape may not set Warp's window title as reported by System Events | Manual testing will verify; Warp documents support for OSC title sequences. If Warp overrides the OSC 0 title with its own auto-generated title, a fallback is to check window content rather than title, or adjust the ANSI escape variant |

## Sources & References

- **Origin document:** [docs/brainstorms/warp-terminal-support-requirements.md](docs/brainstorms/warp-terminal-support-requirements.md)
- Related code: `internal/session/terminal.go` (all existing handlers)
- Warp TERM_PROGRAM value: confirmed as `WarpTerminal` by user
