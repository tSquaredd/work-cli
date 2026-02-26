package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// SessionRecord represents a tracked Claude session.
type SessionRecord struct {
	TaskName      string    `json:"task_name"`
	PID           int       `json:"pid"`
	Dirs          []string  `json:"dirs"`
	LaunchedAt    time.Time `json:"launched_at"`
	TerminalTab   string    `json:"terminal_tab,omitempty"`
	WorkspaceRoot string    `json:"workspace_root"`
}

// Tracker manages session PID files on disk.
type Tracker struct {
	stateDir string // e.g. ~/.local/state/work-cli/sessions/<workspace-hash>/
}

// NewTracker creates a Tracker for the given workspace root.
// Session files are namespaced by a hash of the workspace root to support
// multiple workspaces.
func NewTracker(workspaceRoot string) (*Tracker, error) {
	base := sessionBaseDir()
	hash := hashWorkspace(workspaceRoot)
	dir := filepath.Join(base, hash)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating session state dir: %w", err)
	}

	return &Tracker{stateDir: dir}, nil
}

// Register writes a session PID file for the given task.
func (t *Tracker) Register(rec SessionRecord) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session record: %w", err)
	}
	data = append(data, '\n')

	path := t.pidFilePath(rec.TaskName)
	return os.WriteFile(path, data, 0o644)
}

// Unregister removes the session PID file for a task.
func (t *Tracker) Unregister(taskName string) error {
	path := t.pidFilePath(taskName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Get returns the session record for a task if it exists and the process is alive.
func (t *Tracker) Get(taskName string) (SessionRecord, bool) {
	rec, err := t.read(taskName)
	if err != nil {
		return SessionRecord{}, false
	}

	if !isProcessAlive(rec.PID) {
		// Stale PID file — clean up
		_ = t.Unregister(taskName)
		return SessionRecord{}, false
	}

	return rec, true
}

// IsActive returns true if the given task has a running session.
func (t *Tracker) IsActive(taskName string) bool {
	_, ok := t.Get(taskName)
	return ok
}

// ActiveSessions returns all session records with live processes.
func (t *Tracker) ActiveSessions() []SessionRecord {
	entries, err := os.ReadDir(t.stateDir)
	if err != nil {
		return nil
	}

	var active []SessionRecord
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskName := entry.Name()
		taskName = taskName[:len(taskName)-len(".json")]

		if rec, ok := t.Get(taskName); ok {
			active = append(active, rec)
		}
	}

	return active
}

func (t *Tracker) read(taskName string) (SessionRecord, error) {
	path := t.pidFilePath(taskName)
	data, err := os.ReadFile(path)
	if err != nil {
		return SessionRecord{}, err
	}

	var rec SessionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return SessionRecord{}, err
	}
	return rec, nil
}

func (t *Tracker) pidFilePath(taskName string) string {
	return filepath.Join(t.stateDir, taskName+".json")
}

func sessionBaseDir() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "work-cli", "sessions")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "work-cli", "sessions")
}

func hashWorkspace(root string) string {
	h := sha256.Sum256([]byte(root))
	return hex.EncodeToString(h[:8]) // 16 hex chars is plenty
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// syscall.Kill with signal 0 checks if the process exists
	return syscall.Kill(pid, 0) == nil
}
