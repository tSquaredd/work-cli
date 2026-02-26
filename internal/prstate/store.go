package prstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PRRecord tracks a known PR for a task's worktree.
type PRRecord struct {
	TaskName     string    `json:"task_name"`
	RepoAlias    string    `json:"repo_alias"`
	Number       int       `json:"number"`
	URL          string    `json:"url"`
	LastViewed   time.Time `json:"last_viewed"`
	LastComments int       `json:"last_comments"`
}

// Store manages PR state files on disk, following the session.Tracker pattern.
type Store struct {
	stateDir string // e.g. ~/.local/state/work-cli/prs/<workspace-hash>/
}

// NewStore creates a Store for the given workspace root.
func NewStore(workspaceRoot string) (*Store, error) {
	base := prBaseDir()
	hash := hashWorkspace(workspaceRoot)
	dir := filepath.Join(base, hash)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating pr state dir: %w", err)
	}

	return &Store{stateDir: dir}, nil
}

// Save writes a PR record to disk.
func (s *Store) Save(rec PRRecord) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling pr record: %w", err)
	}
	data = append(data, '\n')

	path := s.filePath(rec.TaskName, rec.RepoAlias)
	return os.WriteFile(path, data, 0o644)
}

// Get returns a PR record for the given task and repo alias, if it exists.
func (s *Store) Get(taskName, repoAlias string) (PRRecord, bool) {
	path := s.filePath(taskName, repoAlias)
	data, err := os.ReadFile(path)
	if err != nil {
		return PRRecord{}, false
	}

	var rec PRRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return PRRecord{}, false
	}
	return rec, true
}

// ForTask returns all PR records for a given task name.
func (s *Store) ForTask(taskName string) []PRRecord {
	entries, err := os.ReadDir(s.stateDir)
	if err != nil {
		return nil
	}

	prefix := taskName + "--"
	var records []PRRecord
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.stateDir, entry.Name()))
		if err != nil {
			continue
		}

		var rec PRRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		records = append(records, rec)
	}
	return records
}

// MarkViewed updates the LastViewed timestamp and LastComments count.
func (s *Store) MarkViewed(taskName, repoAlias string, commentCount int) error {
	rec, ok := s.Get(taskName, repoAlias)
	if !ok {
		return fmt.Errorf("no pr record for %s/%s", taskName, repoAlias)
	}

	rec.LastViewed = time.Now()
	rec.LastComments = commentCount
	return s.Save(rec)
}

// Delete removes the PR record for the given task and repo alias.
func (s *Store) Delete(taskName, repoAlias string) error {
	path := s.filePath(taskName, repoAlias)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) filePath(taskName, repoAlias string) string {
	return filepath.Join(s.stateDir, taskName+"--"+repoAlias+".json")
}

func prBaseDir() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "work-cli", "prs")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "work-cli", "prs")
}

func hashWorkspace(root string) string {
	h := sha256.Sum256([]byte(root))
	return hex.EncodeToString(h[:8])
}
