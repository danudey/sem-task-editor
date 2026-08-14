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

// getGitRemotes simply returns the list of all remote names
func getGitRemotes() ([]string, error) {
	out, err := exec.Command("git", "remote").Output()
	if err != nil {
		return []string{}, fmt.Errorf("failed to get git remotes: %w", err)
	}
	gitRemotes := make([]string, 0)
	for gitRemote := range strings.Lines(string(out)) {
		remoteTrimmed := strings.TrimSpace(gitRemote)
		if remoteTrimmed != "" {
			gitRemotes = append(gitRemotes, remoteTrimmed)
		}
	}

	if len(gitRemotes) == 0 {
		return []string{}, fmt.Errorf("No remotes found in this repository")
	}

	return gitRemotes, nil
}

func gitRemoteForRepoID(expectedRepoID string) (string, error) {
	remotes, err := getGitRemotes()
	if err != nil {
		return "", fmt.Errorf("Could not list git remotes: %w", err)
	}

	for _, remote := range remotes {
		out, err := exec.Command("git", "remote", "get-url", remote).Output()
		if err != nil {
			return "", fmt.Errorf("could not fetch url for remote %s: %w", remote, err)
		}

		repoID := normalizeRepoID(strings.TrimSpace(string(out)))
		if repoID == expectedRepoID {
			return remote, nil
		}
	}
	return "", fmt.Errorf("no remote found with repo ID %s", expectedRepoID)
}

// CheckRepository verifies the current directory is a clone of the expected repo.
func CheckRepository(expectedURL string) error {
	expected := normalizeRepoID(expectedURL)

	_, err := gitRemoteForRepoID(expected)
	if err != nil {
		return fmt.Errorf("unable to locate remote for URL %s", expectedURL)
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
