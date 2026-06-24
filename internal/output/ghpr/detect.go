package ghpr

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// RepoSlug resolves "owner/repo" from the git remote of dir.
// Supports SSH (git@github.com:owner/repo.git) and HTTPS remotes.
func RepoSlug(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w", err)
	}
	return parseRemoteURL(strings.TrimSpace(string(out)))
}

func parseRemoteURL(u string) (string, error) {
	// SSH: git@github.com:owner/repo.git
	if idx := strings.Index(u, "github.com:"); idx >= 0 {
		return strings.TrimSuffix(u[idx+len("github.com:"):], ".git"), nil
	}
	// HTTPS: https://github.com/owner/repo.git  or  github.com/owner/repo
	if idx := strings.Index(u, "github.com/"); idx >= 0 {
		return strings.TrimSuffix(u[idx+len("github.com/"):], ".git"), nil
	}
	return "", fmt.Errorf("not a GitHub remote: %s", u)
}

// OpenPR returns the number of the open PR for the current branch in dir.
// Requires gh CLI to be installed and authenticated.
func OpenPR(dir string) (int, error) {
	cmd := exec.Command("gh", "pr", "view", "--json", "number", "--jq", ".number")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("no open PR found for current branch — set --pr <number> explicitly")
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parsing PR number from gh: %w", err)
	}
	return n, nil
}
