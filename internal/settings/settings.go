// Package settings persists user-level preferences for work-cli at
// ~/.config/work-cli/config.json. The first preference governs whether to pass
// --dangerously-skip-permissions when launching Claude.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	DangerouslySkipAsk    = "ask"
	DangerouslySkipAlways = "always"
	DangerouslySkipNever  = "never"
)

// Settings holds user-level preferences for work-cli.
type Settings struct {
	DangerouslySkipPermissions string `json:"dangerously_skip_permissions"`
}

// Default returns the baseline settings used when no config file exists.
func Default() Settings {
	return Settings{
		DangerouslySkipPermissions: DangerouslySkipAsk,
	}
}

// Path returns the on-disk location of the settings file.
func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "work-cli", "config.json")
}

// Load reads settings from disk. A missing file is not an error — Default() is
// returned. Parse errors return Default() alongside the error so callers can
// degrade gracefully while surfacing the problem.
func Load() (Settings, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}
		return Default(), fmt.Errorf("reading settings: %w", err)
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Default(), fmt.Errorf("parsing settings: %w", err)
	}
	normalize(&s)
	return s, nil
}

// Save writes settings atomically (tmp file + rename).
func Save(s Settings) error {
	normalize(&s)

	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), "config.json.*")
	if err != nil {
		return fmt.Errorf("creating tmp settings file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing settings tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming settings into place: %w", err)
	}
	return nil
}

// ResolveDangerouslySkip returns whether the launch site should prompt the user
// and, if not, the value to use directly.
func ResolveDangerouslySkip(s Settings) (prompt bool, value bool) {
	switch s.DangerouslySkipPermissions {
	case DangerouslySkipAlways:
		return false, true
	case DangerouslySkipNever:
		return false, false
	default:
		return true, false
	}
}

func normalize(s *Settings) {
	switch s.DangerouslySkipPermissions {
	case DangerouslySkipAsk, DangerouslySkipAlways, DangerouslySkipNever:
		// valid
	default:
		s.DangerouslySkipPermissions = DangerouslySkipAsk
	}
}
