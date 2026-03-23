package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// normalizeRepoID extracts "owner/repo" from a git URL.
// Handles both SSH (git@github.com:owner/repo.git) and HTTPS (https://github.com/owner/repo.git).
func normalizeRepoID(url string) string {
	url = strings.TrimSuffix(url, ".git")

	// SSH format: git@github.com:owner/repo
	if strings.Contains(url, ":") && strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// HTTPS format: https://github.com/owner/repo
	re := regexp.MustCompile(`(?:https?://[^/]+/)(.+)`)
	if m := re.FindStringSubmatch(url); len(m) == 2 {
		return m[1]
	}

	return url
}

// CheckRepository verifies the current directory is a clone of the expected repo.
func CheckRepository(expectedURL string) error {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return fmt.Errorf("not a git repository or no 'origin' remote: %w", err)
	}

	actualURL := strings.TrimSpace(string(out))
	expected := normalizeRepoID(expectedURL)
	actual := normalizeRepoID(actualURL)

	if expected != actual {
		return fmt.Errorf("repository mismatch: expected %s (%s), got %s (%s)",
			expected, expectedURL, actual, actualURL)
	}

	return nil
}

// ListBranches returns the list of branch names from the remote.
func ListBranches() ([]string, error) {
	out, err := exec.Command("git", "ls-remote", "--heads", "origin").Output()
	if err != nil {
		return nil, fmt.Errorf("listing remote branches: %w", err)
	}

	var branches []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ref := parts[1]
			branch := strings.TrimPrefix(ref, "refs/heads/")
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

// FileCheckResult indicates whether a file exists in a branch.
type FileCheckResult struct {
	Exists       bool
	RefAvailable bool
}

// FileExistsInBranch checks if a file path exists in the given branch.
// Returns whether the file exists and whether the branch ref was available locally.
func FileExistsInBranch(branch, path string) FileCheckResult {
	ref := "origin/" + branch

	// Check if the ref exists locally
	if err := exec.Command("git", "rev-parse", "--verify", ref).Run(); err != nil {
		return FileCheckResult{Exists: false, RefAvailable: false}
	}

	// Check if the file exists in the branch
	err := exec.Command("git", "cat-file", "-e", ref+":"+path).Run()
	return FileCheckResult{Exists: err == nil, RefAvailable: true}
}
