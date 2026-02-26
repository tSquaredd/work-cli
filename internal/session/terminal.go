package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TabOpener opens a new terminal tab and runs a command in it.
type TabOpener interface {
	OpenTab(command, title string) (pid int, err error)
	FocusTab(identifier string) error
}

// DetectTerminal returns a TabOpener for the current terminal environment.
// Falls back to a no-op opener that prints the command for manual execution.
func DetectTerminal() TabOpener {
	term := os.Getenv("TERM_PROGRAM")
	switch strings.ToLower(term) {
	case "iterm.app", "iterm2":
		return &iterm2Opener{}
	case "apple_terminal":
		return &terminalAppOpener{}
	default:
		return &fallbackOpener{}
	}
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

	// PID capture is imprecise with AppleScript — return 0 to indicate launched without PID
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
