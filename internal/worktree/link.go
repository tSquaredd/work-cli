package worktree

import (
	"os"
	"path/filepath"
	"strings"
)

// LinkResult describes what was linked.
type LinkResult struct {
	RepoName string
	Files    []string
}

// LinkBuildFiles symlinks gitignored build-essential files from the main repo
// into a worktree. Detects project type: Gradle → local.properties, Node → .env*.
func LinkBuildFiles(wtDir, mainDir string) LinkResult {
	result := LinkResult{RepoName: filepath.Base(mainDir)}

	// Gradle projects: symlink local.properties (root + one level deep)
	if fileExists(filepath.Join(mainDir, "gradlew")) {
		if linked := symlinkIfNeeded(
			filepath.Join(mainDir, "local.properties"),
			filepath.Join(wtDir, "local.properties"),
		); linked {
			result.Files = append(result.Files, "local.properties")
		}

		// One level deep
		entries, err := os.ReadDir(mainDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				src := filepath.Join(mainDir, e.Name(), "local.properties")
				dst := filepath.Join(wtDir, e.Name(), "local.properties")
				if fileExists(src) && isDir(filepath.Join(wtDir, e.Name())) {
					if symlinkIfNeeded(src, dst) {
						result.Files = append(result.Files, e.Name()+"/local.properties")
					}
				}
			}
		}
	}

	// Node projects: symlink .env* files
	if fileExists(filepath.Join(mainDir, "package.json")) {
		entries, err := os.ReadDir(mainDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if !strings.HasPrefix(name, ".env") {
					continue
				}
				if name == ".env.example" {
					continue
				}
				src := filepath.Join(mainDir, name)
				dst := filepath.Join(wtDir, name)
				if symlinkIfNeeded(src, dst) {
					result.Files = append(result.Files, name)
				}
			}
		}
	}

	return result
}

func symlinkIfNeeded(src, dst string) bool {
	if !fileExists(src) {
		return false
	}
	if _, err := os.Lstat(dst); err == nil {
		return false // Already exists
	}
	if err := os.Symlink(src, dst); err != nil {
		return false
	}
	return true
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
