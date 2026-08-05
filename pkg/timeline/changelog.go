package timeline

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Release struct {
	Version string
	Date    time.Time
	Notes   string
}

// ParseChangelog reads a Keep-a-Changelog file: each `## [1.2.3] - 2026-01-02`
// heading opens a release whose body runs to the next `## ` heading. Sections
// without a date fall back to the file's own modification time, and an
// `Unreleased` section is skipped because it has no version to key on.
func ParseChangelog(path string) ([]Release, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read changelog %q: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat changelog %q: %w", path, err)
	}

	lines := strings.Split(string(raw), "\n")
	releases := []Release{}
	var current *Release
	var body []string

	flush := func() {
		if current == nil {
			return
		}
		current.Notes = strings.TrimSpace(strings.Join(body, "\n"))
		releases = append(releases, *current)
		current = nil
		body = nil
	}

	for _, line := range lines {
		match := changelogRe.FindStringSubmatch(line)
		if match == nil {
			if current != nil {
				body = append(body, line)
			}
			continue
		}

		flush()

		date := info.ModTime()
		if match[2] != "" {
			parsed, err := parseDate(match[2])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			date = parsed
		}
		current = &Release{Version: match[1], Date: date}
	}
	flush()

	return releases, nil
}

// ChangelogNotes returns the body of the section matching version, ignoring a
// leading "v" on either side so `v1.2.3` finds `## [1.2.3]`.
func ChangelogNotes(path, version string) (string, error) {
	releases, err := ParseChangelog(path)
	if err != nil {
		return "", err
	}
	want := strings.TrimPrefix(version, "v")
	for _, release := range releases {
		if strings.TrimPrefix(release.Version, "v") == want {
			return release.Notes, nil
		}
	}
	return "", nil
}
