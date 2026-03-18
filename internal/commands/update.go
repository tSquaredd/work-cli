package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
	"github.com/tSquaredd/work-cli/internal/ui"
)

const (
	updateRepo     = "tSquaredd/work-cli"
	cacheDir       = ".cache/work-cli"
	cacheFile      = "latest-version"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Self-update to the latest version from GitHub",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate()
		},
	}
}

func runUpdate() error {
	fmt.Println()
	fmt.Println(ui.Section("Checking for updates..."))
	fmt.Println()

	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(updateRepo))
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if !found {
		fmt.Println(ui.StyleDim.Render("  No releases found."))
		return nil
	}

	currentVersion := version
	if strings.HasPrefix(currentVersion, "v") {
		currentVersion = currentVersion[1:]
	}
	latestVersion := latest.Version()

	if latestVersion == currentVersion {
		fmt.Printf("  %s (v%s)\n", ui.StyleSuccess.Render("Already up to date"), currentVersion)
		clearVersionCache()
		fmt.Println()
		return nil
	}

	fmt.Printf("  Updating v%s → v%s...\n", currentVersion, latestVersion)
	fmt.Println()

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("updating: %w", err)
	}

	clearVersionCache()

	fmt.Printf("  %s v%s → v%s\n", ui.StyleSuccess.Render("Updated:"), currentVersion, latestVersion)
	fmt.Println()
	return nil
}

// checkForUpdateBg checks for updates in the background.
// Returns the newer version string if one is cached, or empty string if up to date.
func checkForUpdateBg() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	cachePath := filepath.Join(home, cacheDir, cacheFile)

	var availableVersion string
	if data, err := os.ReadFile(cachePath); err == nil {
		cached := strings.TrimSpace(string(data))
		currentVersion := version
		if strings.HasPrefix(currentVersion, "v") {
			currentVersion = currentVersion[1:]
		}
		if cached != "" && cached != currentVersion {
			availableVersion = cached
		}
	}

	// Background fetch
	go func() {
		latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(updateRepo))
		if err != nil || !found {
			return
		}

		dir := filepath.Join(home, cacheDir)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, cacheFile), []byte(latest.Version()), 0o644)
	}()

	return availableVersion
}

func clearVersionCache() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	_ = os.Remove(filepath.Join(home, cacheDir, cacheFile))
}
