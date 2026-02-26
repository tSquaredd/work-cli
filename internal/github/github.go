package github

import (
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// repoCache caches owner/repo per directory to avoid redundant calls.
var (
	repoCache   = make(map[string][2]string)
	repoCacheMu sync.Mutex
)

// IsAvailable returns true if the gh CLI is installed and authenticated.
func IsAvailable() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

// RepoFromRemote parses the GitHub owner and repo from the git remote URL.
// It caches results per directory.
func RepoFromRemote(dir string) (owner, repo string, err error) {
	repoCacheMu.Lock()
	if cached, ok := repoCache[dir]; ok {
		repoCacheMu.Unlock()
		return cached[0], cached[1], nil
	}
	repoCacheMu.Unlock()

	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	owner, repo = parseGitHubURL(strings.TrimSpace(string(out)))
	if owner == "" {
		return "", "", nil
	}

	repoCacheMu.Lock()
	repoCache[dir] = [2]string{owner, repo}
	repoCacheMu.Unlock()

	return owner, repo, nil
}

// parseGitHubURL extracts owner and repo from SSH or HTTPS GitHub URLs.
// Supports:
//   - git@github.com:owner/repo.git
//   - https://github.com/owner/repo.git
//   - https://github.com/owner/repo
func parseGitHubURL(url string) (owner, repo string) {
	// SSH: git@github.com:owner/repo.git
	sshRe := regexp.MustCompile(`github\.com[:/]([^/]+)/([^/]+?)(?:\.git)?$`)
	if m := sshRe.FindStringSubmatch(url); len(m) == 3 {
		return m[1], m[2]
	}
	return "", ""
}
