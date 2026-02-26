package workspace

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AutoPrefix returns a short branch prefix based on the directory name.
func AutoPrefix(name string) string {
	lower := strings.ToLower(name)

	switch {
	case strings.Contains(lower, "android"):
		return "and"
	case strings.Contains(lower, "ios"):
		return "ios"
	case strings.Contains(lower, "kmp") ||
		strings.Contains(lower, "multiplatform") ||
		strings.Contains(lower, "shared"):
		return "kmp"
	case strings.Contains(lower, "web") ||
		strings.Contains(lower, "frontend"):
		return "web"
	case strings.Contains(lower, "backend") ||
		strings.Contains(lower, "server") ||
		strings.Contains(lower, "api"):
		return "api"
	default:
		re := regexp.MustCompile(`[^a-z0-9]`)
		clean := re.ReplaceAllString(lower, "")
		if len(clean) > 3 {
			clean = clean[:3]
		}
		return clean
	}
}

// AutoDescription detects a project description from repo contents.
func AutoDescription(dir string) string {
	// iOS
	if hasGlob(dir, "*.xcodeproj") || hasGlob(dir, "*.xcworkspace") {
		return "iOS (Swift)"
	}

	// Gradle projects
	if fileExists(filepath.Join(dir, "gradlew")) {
		if fileExists(filepath.Join(dir, "shared", "build.gradle.kts")) {
			return "Kotlin Multiplatform"
		}
		if containsAndroid(dir) {
			return "Android (Kotlin)"
		}
		return "Kotlin/Gradle"
	}

	// Node projects
	if fileExists(filepath.Join(dir, "package.json")) {
		if fileExists(filepath.Join(dir, ".meteor", "release")) {
			return "Meteor + React"
		}
		if fileExists(filepath.Join(dir, "next.config.js")) ||
			fileExists(filepath.Join(dir, "next.config.mjs")) ||
			fileExists(filepath.Join(dir, "next.config.ts")) {
			return "Next.js"
		}
		if fileExists(filepath.Join(dir, "vite.config.ts")) ||
			fileExists(filepath.Join(dir, "vite.config.js")) {
			return "Vite"
		}
		return "JavaScript/TypeScript"
	}

	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return "Rust"
	}
	if fileExists(filepath.Join(dir, "go.mod")) {
		return "Go"
	}
	if fileExists(filepath.Join(dir, "pyproject.toml")) || fileExists(filepath.Join(dir, "requirements.txt")) {
		return "Python"
	}
	if fileExists(filepath.Join(dir, "Gemfile")) {
		return "Ruby"
	}

	return "Git repository"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasGlob(dir, pattern string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	return err == nil && len(matches) > 0
}

func containsAndroid(dir string) bool {
	// Check directory name
	if strings.Contains(strings.ToLower(filepath.Base(dir)), "android") {
		return true
	}
	// Quick check for AndroidManifest.xml one level deep
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifest := filepath.Join(dir, e.Name(), "src", "main", "AndroidManifest.xml")
		if fileExists(manifest) {
			return true
		}
	}
	return false
}
