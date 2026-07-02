package git

import (
	"os/exec"
	"strings"
)

// Metadata represents git repository metadata
type Metadata struct {
	CommitHash string `json:"commitHash"`
	Branch     string `json:"branch"`
	IsDirty    bool   `json:"isDirty"`
	RemoteURL  string `json:"remoteUrl,omitempty"`
}

// CaptureMetadata captures git metadata from the current repository
// If git is unavailable or any command fails, it returns partial data
func CaptureMetadata() Metadata {
	meta := Metadata{}

	// Capture commit hash
	if hash, err := runGitCommand("rev-parse", "HEAD"); err == nil {
		meta.CommitHash = strings.TrimSpace(hash)
	}

	// Capture branch
	if branch, err := runGitCommand("rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		meta.Branch = strings.TrimSpace(branch)
	}

	// Check if repository is dirty
	if status, err := runGitCommand("status", "--porcelain"); err == nil {
		meta.IsDirty = strings.TrimSpace(status) != ""
	}

	// Capture remote URL (try origin first)
	if remoteURL, err := runGitCommand("config", "--get", "remote.origin.url"); err == nil {
		meta.RemoteURL = strings.TrimSpace(remoteURL)
	}

	return meta
}

// runGitCommand executes a git command and returns its output
func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
