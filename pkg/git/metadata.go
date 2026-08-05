package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
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

// FileAddedAt reports when a file first entered the repository, used as the
// event date for ADRs and postmortems that carry no explicit date.
func FileAddedAt(path string) (time.Time, error) {
	out, err := runGitCommand("log", "--diff-filter=A", "--format=%aI", "-1", "--", path)
	if err != nil {
		return time.Time{}, err
	}
	raw := strings.TrimSpace(out)
	if raw == "" {
		return time.Time{}, fmt.Errorf("no add commit found for %s", path)
	}
	return time.Parse(time.RFC3339, raw)
}

// TagAtHead returns the tag pointing exactly at HEAD, which is what a
// tag-triggered pipeline has checked out.
func TagAtHead() (string, error) {
	out, err := runGitCommand("describe", "--tags", "--exact-match", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// AnnotatedTagBody returns the message body of an annotated tag, empty for a
// lightweight tag.
func AnnotatedTagBody(tag string) (string, error) {
	out, err := runGitCommand("tag", "-l", "--format=%(contents:body)", tag)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// PreviousTag returns the most recent tag reachable from the commit before tag.
func PreviousTag(tag string) (string, error) {
	out, err := runGitCommand("describe", "--tags", "--abbrev=0", tag+"^")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// CommitsBetween lists "<short hash> <subject>" for each commit in from..to.
// An empty from means every commit reachable from to.
func CommitsBetween(from, to string) ([]string, error) {
	revRange := to
	if from != "" {
		revRange = from + ".." + to
	}
	out, err := runGitCommand("log", "--pretty=%h %s", revRange)
	if err != nil {
		return nil, err
	}
	commits := []string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		commits = append(commits, trimmed)
	}
	return commits, nil
}

// TagDate returns the tagger date of an annotated tag, falling back to the
// commit date for a lightweight tag.
func TagDate(tag string) (time.Time, error) {
	out, err := runGitCommand("log", "-1", "--format=%aI", tag)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(out))
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
