# Warp Terminal Support

**Date:** 2026-04-16
**Status:** Ready for planning
**Scope:** Lightweight

## Problem

`work-cli` manages parallel Claude Code sessions by opening new terminal tabs and focusing existing ones. It currently supports Ghostty, iTerm2, and Terminal.app. Users running Warp terminal fall through to the `fallbackOpener`, which prints the command for manual execution — breaking the core UX of launching and resuming tasks from the dashboard.

## Goal

Warp users should get the same tab-launch and tab-focus experience that Ghostty users have today.

## Requirements

1. **Detection**: When `TERM_PROGRAM=WarpTerminal`, `DetectTerminal()` returns a `warpOpener`.
2. **OpenTab**: Opens a new Warp tab and executes the provided command. Uses AppleScript UI scripting via System Events (Cmd+T to open tab, keystroke to enter command), same pattern as `ghosttyOpener`. Sets tab title via ANSI escape (`\033]0;title\007`).
3. **FocusTab**: Activates Warp and cycles through tabs checking window title for the identifier, same approach as Ghostty's `FocusTab`.
4. **Command file pattern**: Uses the existing `writeCommandFile()` helper to avoid AppleScript escaping issues.

## Non-goals

- Warp CLI cold-start (no `-e` flag equivalent exists)
- Warp URI scheme integration (`warp://action/new_tab` can't execute commands)
- Warp Tab Configs (TOML files can't be triggered programmatically)
- Adding tests (no test files exist in the project today)

## Approach

Mirror the `ghosttyOpener` implementation with Warp-specific adjustments:
- Process name in System Events: `"Warp"`
- No CLI fallback — AppleScript-only (fallback to `fallbackOpener` behavior if Warp isn't running is acceptable)
- Tab cycling for `FocusTab` uses the same Cmd+Shift+] pattern

## Known Limitations

- **Focus steal on launch**: Warp activates (comes to foreground) when opening a tab, same as Ghostty. iTerm2 avoids this via its native scripting API.
- **Tab focus is approximate**: Cycling through tabs is visible to the user. Works but less polished than iTerm2's direct tab selection.
