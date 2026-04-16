package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// writeCommandFile writes command to a temp script and returns a short shell
// snippet that sources and self-deletes it. This avoids AppleScript escaping
// and length issues that corrupt $$, long --append-system-prompt flags, etc.
func writeCommandFile(command string) (execCmd string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "work-cli-cmd-*.sh")
	if err != nil {
		return "", nil, fmt.Errorf("create temp command file: %w", err)
	}
	path := f.Name()
	cleanup = func() { os.Remove(path) }

	if _, err := f.WriteString(command); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("write temp command file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close temp command file: %w", err)
	}

	execCmd = fmt.Sprintf(`. "%s"; rm -f "%s"`, path, path)
	return execCmd, cleanup, nil
}

// TabOpener opens a new terminal window/tab and runs a command in it.
type TabOpener interface {
	OpenTab(command, title string) (pid int, err error)
	FocusTab(identifier string) error
}

// DetectTerminal returns a TabOpener for the current terminal environment.
// Falls back to a no-op opener that prints the command for manual execution.
func DetectTerminal() TabOpener {
	term := os.Getenv("TERM_PROGRAM")
	switch strings.ToLower(term) {
	case "ghostty":
		return &ghosttyOpener{}
	case "iterm.app", "iterm2":
		return &iterm2Opener{}
	case "warpterminal":
		return &warpOpener{}
	case "apple_terminal":
		return &terminalAppOpener{}
	default:
		return &fallbackOpener{}
	}
}

// ghosttyOpener opens windows in Ghostty via AppleScript (running) or CLI (cold start).
type ghosttyOpener struct{}

func (o *ghosttyOpener) OpenTab(command, title string) (int, error) {
	fullCmd := fmt.Sprintf("printf '\\033]0;%s\\007' && %s", title, command)

	// Try AppleScript first (new window in running instance).
	// This avoids case-sensitivity issues with pgrep — macOS may report
	// the process as "ghostty" (lowercase) while pgrep -x "Ghostty" fails.
	pid, err := o.openViaAppleScript(fullCmd)
	if err == nil {
		return pid, nil
	}
	// AppleScript failed (Ghostty not running) — cold start via CLI
	return o.openViaCLI(fullCmd)
}

// openViaAppleScript opens a new window in the running Ghostty instance.
func (o *ghosttyOpener) openViaAppleScript(fullCmd string) (int, error) {
	execCmd, cleanup, err := writeCommandFile(fullCmd)
	if err != nil {
		return 0, fmt.Errorf("Ghostty command file: %w", err)
	}

	script := fmt.Sprintf(`
tell application "System Events"
	if not (exists process "Ghostty") then
		error "Ghostty not running"
	end if
	tell process "Ghostty"
		set frontmost to true
	end tell
end tell
delay 0.3
tell application "System Events"
	tell process "Ghostty"
		keystroke "t" using command down
	end tell
end tell
delay 0.5
tell application "System Events"
	tell process "Ghostty"
		keystroke %q
		key code 36
	end tell
end tell
`, execCmd)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		cleanup() // remove temp file since terminal never got the command
		return 0, fmt.Errorf("Ghostty AppleScript failed: %w", err)
	}
	// temp file self-deletes via "rm -f" in the exec snippet
	return 0, nil
}

// openViaCLI cold-starts Ghostty via its CLI binary.
func (o *ghosttyOpener) openViaCLI(fullCmd string) (int, error) {
	ghosttyPath, _ := exec.LookPath("ghostty")
	if ghosttyPath == "" {
		appPath := "/Applications/Ghostty.app/Contents/MacOS/ghostty"
		if _, err := os.Stat(appPath); err == nil {
			ghosttyPath = appPath
		}
	}

	if ghosttyPath == "" {
		return 0, fmt.Errorf("Ghostty not found in PATH or /Applications")
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	cmd := exec.Command(ghosttyPath, "-e", shell, "-c", fullCmd)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("Ghostty CLI launch failed: %w", err)
	}
	return cmd.Process.Pid, nil
}

func (o *ghosttyOpener) FocusTab(identifier string) error {
	script := fmt.Sprintf(`
tell application "System Events"
	if not (exists process "Ghostty") then
		error "Ghostty not running"
	end if
	tell process "Ghostty"
		set frontmost to true
	end tell
end tell
delay 0.3
tell application "System Events"
	tell process "Ghostty"
		set maxTabs to 20
		repeat maxTabs times
			set winTitle to name of front window
			if winTitle contains %q then
				return
			end if
			-- Cycle to next tab: Cmd+Shift+]
			key code 30 using {command down, shift down}
			delay 0.1
		end repeat
	end tell
end tell
`, identifier)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// warpOpener opens tabs in Warp via AppleScript UI scripting (System Events).
// Warp has no AppleScript dictionary and no CLI launcher flag (unlike Ghostty's
// -e), so OpenTab requires a running Warp instance. TERM_PROGRAM=WarpTerminal
// is only set inside a running Warp session, so callers will almost always
// have one, but we still surface a clear error if the process disappears.
type warpOpener struct{}

// warpActivateScript polls for Warp to become frontmost before sending
// keystrokes. A fixed `delay` is racy — under load, keystrokes can arrive
// before Warp has focus and get sent to the wrong app. The poll caps at ~2s;
// after that AppleScript errors out rather than silently dropping input.
const warpActivateScript = `
tell application "System Events"
	if not (exists process "Warp") then
		error "Warp not running"
	end if
	tell process "Warp" to set frontmost to true
	set waited to 0
	repeat until (frontmost of process "Warp") or waited >= 20
		delay 0.1
		set waited to waited + 1
	end repeat
	if not (frontmost of process "Warp") then
		error "Warp did not come to front"
	end if
end tell`

// interpretWarpOsaError converts osascript output into a targeted error.
// macOS emits -1719 ("not allowed assistive access") when the calling terminal
// lacks Accessibility permission; without this check the user sees only a
// cryptic "osascript: exit status 1".
func interpretWarpOsaError(out []byte, err error) error {
	s := string(out)
	if strings.Contains(s, "-1719") || strings.Contains(strings.ToLower(s), "not allowed assistive access") {
		return fmt.Errorf(
			"Warp tab open blocked: the terminal running `work` needs Accessibility permission. " +
				"Open System Settings → Privacy & Security → Accessibility and enable your terminal app, then retry")
	}
	if trimmed := strings.TrimSpace(s); trimmed != "" {
		return fmt.Errorf("Warp AppleScript failed: %w: %s", err, trimmed)
	}
	return fmt.Errorf("Warp AppleScript failed: %w", err)
}

func (o *warpOpener) OpenTab(command, title string) (int, error) {
	fullCmd := fmt.Sprintf("printf '\\033]0;%s\\007' && %s", title, command)

	execCmd, cleanup, err := writeCommandFile(fullCmd)
	if err != nil {
		return 0, fmt.Errorf("Warp command file: %w", err)
	}

	script := fmt.Sprintf(`%s
tell application "System Events"
	tell process "Warp" to keystroke "t" using command down
end tell
delay 0.3
tell application "System Events"
	tell process "Warp"
		keystroke %q
		key code 36
	end tell
end tell
`, warpActivateScript, execCmd)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cleanup() // remove temp file since terminal never got the command
		return 0, interpretWarpOsaError(out, err)
	}
	// temp file self-deletes via "rm -f" in the exec snippet
	return 0, nil
}

func (o *warpOpener) FocusTab(identifier string) error {
	script := fmt.Sprintf(`%s
tell application "System Events"
	tell process "Warp"
		set maxTabs to 20
		repeat maxTabs times
			set winTitle to name of front window
			if winTitle contains %q then
				return
			end if
			-- Cycle to next tab: Cmd+Shift+]
			key code 30 using {command down, shift down}
			delay 0.1
		end repeat
	end tell
end tell
`, warpActivateScript, identifier)
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return interpretWarpOsaError(out, err)
	}
	return nil
}

// iterm2Opener opens tabs in iTerm2 via AppleScript.
type iterm2Opener struct{}

func (o *iterm2Opener) OpenTab(command, title string) (int, error) {
	execCmd, cleanup, err := writeCommandFile(command)
	if err != nil {
		return 0, fmt.Errorf("iTerm2 command file: %w", err)
	}

	script := fmt.Sprintf(`
tell application "iTerm2"
	tell current window
		create tab with default profile
		tell current session
			set name to %q
			write text %q
		end tell
	end tell
end tell
`, title, execCmd)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		cleanup()
		return 0, fmt.Errorf("iTerm2 AppleScript failed: %w", err)
	}

	return 0, nil
}

func (o *iterm2Opener) FocusTab(identifier string) error {
	script := fmt.Sprintf(`
tell application "iTerm2"
	activate
	repeat with w in windows
		tell w
			repeat with t in tabs
				tell t
					repeat with s in sessions
						if name of s contains %q then
							select t
							return
						end if
					end repeat
				end tell
			end repeat
		end tell
	end repeat
end tell
`, identifier)

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// terminalAppOpener opens tabs in Terminal.app via AppleScript.
type terminalAppOpener struct{}

func (o *terminalAppOpener) OpenTab(command, title string) (int, error) {
	execCmd, cleanup, err := writeCommandFile(command)
	if err != nil {
		return 0, fmt.Errorf("Terminal.app command file: %w", err)
	}

	script := fmt.Sprintf(`
tell application "Terminal"
	activate
	tell application "System Events"
		keystroke "t" using command down
	end tell
	delay 0.5
	do script %q in front window
	set custom title of front window to %q
end tell
`, execCmd, title)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		cleanup()
		return 0, fmt.Errorf("Terminal.app AppleScript failed: %w", err)
	}

	return 0, nil
}

func (o *terminalAppOpener) FocusTab(identifier string) error {
	script := fmt.Sprintf(`
tell application "Terminal"
	activate
	repeat with w in windows
		if custom title of w contains %q then
			set index of w to 1
			return
		end if
	end repeat
end tell
`, identifier)

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// fallbackOpener is used when the terminal is not recognized.
// It prints the command for manual execution.
type fallbackOpener struct{}

func (o *fallbackOpener) OpenTab(command, title string) (int, error) {
	return 0, fmt.Errorf("unsupported terminal — run manually:\n  %s", command)
}

func (o *fallbackOpener) FocusTab(identifier string) error {
	return fmt.Errorf("unsupported terminal — cannot focus tab %q", identifier)
}
