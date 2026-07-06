package git

import (
	"os/exec"
	"strings"
)

type Metadata struct {
	CommitHash string `json:"commitHash"`
	Branch     string `json:"branch"`
	IsDirty    bool   `json:"isDirty"`
	RemoteURL  string `json:"remoteUrl,omitempty"`
}

func CaptureMetadata() Metadata {
	meta := Metadata{}

	if hash, err := runGitCommand("rev-parse", "HEAD"); err == nil {
		meta.CommitHash = strings.TrimSpace(hash)
	}

	if branch, err := runGitCommand("rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		meta.Branch = strings.TrimSpace(branch)
	}

	if status, err := runGitCommand("status", "--porcelain"); err == nil {
		meta.IsDirty = strings.TrimSpace(status) != ""
	}

	if remoteURL, err := runGitCommand("config", "--get", "remote.origin.url"); err == nil {
		meta.RemoteURL = strings.TrimSpace(remoteURL)
	}

	return meta
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
