package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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

	if isGhosttyRunning() {
		return o.openViaAppleScript(fullCmd)
	}
	return o.openViaCLI(fullCmd)
}

// isGhosttyRunning checks if a Ghostty process is already running.
func isGhosttyRunning() bool {
	return exec.Command("pgrep", "-x", "Ghostty").Run() == nil
}

// openViaAppleScript opens a new window in the running Ghostty instance.
func (o *ghosttyOpener) openViaAppleScript(fullCmd string) (int, error) {
	script := fmt.Sprintf(`
tell application "Ghostty"
	activate
end tell
delay 0.3
tell application "System Events"
	tell process "Ghostty"
		click menu item "New Window" of menu "File" of menu bar 1
	end tell
end tell
delay 0.5
tell application "System Events"
	tell process "Ghostty"
		keystroke %q
		key code 36
	end tell
end tell
`, fullCmd)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("Ghostty AppleScript failed: %w", err)
	}
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
	script := `
tell application "Ghostty"
	activate
end tell
`
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// iterm2Opener opens tabs in iTerm2 via AppleScript.
type iterm2Opener struct{}

func (o *iterm2Opener) OpenTab(command, title string) (int, error) {
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
`, title, command)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
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
`, command, title)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
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
