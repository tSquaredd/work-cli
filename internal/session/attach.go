package session

import "fmt"

// Attach focuses the terminal tab running the given task's session.
// Returns an error if the task has no active session or the terminal
// does not support tab focusing.
func Attach(tracker *Tracker, taskName string) error {
	rec, ok := tracker.Get(taskName)
	if !ok {
		return fmt.Errorf("no active session for task %q", taskName)
	}

	tabID := rec.TerminalTab
	if tabID == "" {
		tabID = "work: " + taskName
	}

	opener := DetectTerminal()
	return opener.FocusTab(tabID)
}
