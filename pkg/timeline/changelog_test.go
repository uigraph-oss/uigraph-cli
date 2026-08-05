package timeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const keepAChangelog = `# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

- Nothing yet.

## [1.2.0] - 2026-04-01

### Added

- Cost tag sync from the CLI.

### Fixed

- Timeline events no longer duplicate on re-scan.

## [1.1.0] - 2026-03-02

### Added

- Saved query sync.

## 1.0.0 - 2026-01-15

Initial release.
`

func writeChangelog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseChangelog(t *testing.T) {
	releases, err := ParseChangelog(writeChangelog(t, keepAChangelog))
	if err != nil {
		t.Fatalf("ParseChangelog() error = %v", err)
	}

	if len(releases) != 3 {
		t.Fatalf("len(releases) = %d, want 3 (Unreleased has no version to key on)", len(releases))
	}

	if releases[0].Version != "1.2.0" {
		t.Errorf("releases[0].Version = %q, want 1.2.0", releases[0].Version)
	}
	if releases[0].Date.Format("2006-01-02") != "2026-04-01" {
		t.Errorf("releases[0].Date = %v, want 2026-04-01", releases[0].Date)
	}
	if !strings.Contains(releases[0].Notes, "Cost tag sync from the CLI.") {
		t.Errorf("releases[0].Notes = %q, want the Added section body", releases[0].Notes)
	}
	if strings.Contains(releases[0].Notes, "Saved query sync") {
		t.Errorf("releases[0].Notes bled into the next release: %q", releases[0].Notes)
	}

	if releases[2].Version != "1.0.0" {
		t.Errorf("releases[2].Version = %q, want 1.0.0 from an unbracketed heading", releases[2].Version)
	}
	if releases[2].Notes != "Initial release." {
		t.Errorf("releases[2].Notes = %q, want Initial release.", releases[2].Notes)
	}
}

func TestParseChangelog_VPrefixedHeading(t *testing.T) {
	releases, err := ParseChangelog(writeChangelog(t, "## [v2.0.0] - 2026-05-05\n\nBig one.\n"))
	if err != nil {
		t.Fatalf("ParseChangelog() error = %v", err)
	}
	if len(releases) != 1 || releases[0].Version != "2.0.0" {
		t.Fatalf("releases = %+v, want a single 2.0.0", releases)
	}
}

func TestParseChangelog_MissingDateFallsBackToModTime(t *testing.T) {
	releases, err := ParseChangelog(writeChangelog(t, "## [3.0.0]\n\nNo date here.\n"))
	if err != nil {
		t.Fatalf("ParseChangelog() error = %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("len(releases) = %d, want 1", len(releases))
	}
	if releases[0].Date.IsZero() {
		t.Fatal("Date is zero, want the changelog's mtime")
	}
}

func TestParseChangelog_EmptyFile(t *testing.T) {
	releases, err := ParseChangelog(writeChangelog(t, "# Changelog\n\nNothing released yet.\n"))
	if err != nil {
		t.Fatalf("ParseChangelog() error = %v", err)
	}
	if len(releases) != 0 {
		t.Fatalf("len(releases) = %d, want 0", len(releases))
	}
}

func TestChangelogNotes(t *testing.T) {
	path := writeChangelog(t, keepAChangelog)

	notes, err := ChangelogNotes(path, "1.1.0")
	if err != nil {
		t.Fatalf("ChangelogNotes() error = %v", err)
	}
	if !strings.Contains(notes, "Saved query sync.") {
		t.Errorf("notes = %q, want the 1.1.0 body", notes)
	}
}

func TestChangelogNotes_IgnoresVPrefix(t *testing.T) {
	path := writeChangelog(t, keepAChangelog)

	notes, err := ChangelogNotes(path, "v1.1.0")
	if err != nil {
		t.Fatalf("ChangelogNotes() error = %v", err)
	}
	if !strings.Contains(notes, "Saved query sync.") {
		t.Errorf("notes = %q, want v1.1.0 to find the [1.1.0] section", notes)
	}
}

func TestChangelogNotes_UnknownVersionIsEmpty(t *testing.T) {
	path := writeChangelog(t, keepAChangelog)

	notes, err := ChangelogNotes(path, "9.9.9")
	if err != nil {
		t.Fatalf("ChangelogNotes() error = %v", err)
	}
	if notes != "" {
		t.Errorf("notes = %q, want empty so the caller falls through to commit subjects", notes)
	}
}

func TestScan_ChangelogBecomesReleaseEvents(t *testing.T) {
	path := writeChangelog(t, keepAChangelog)

	events, err := Scan(scanConfig(filepath.Dir(path)))
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0 when changelogPath is unset", len(events))
	}

	cfg := scanConfig(filepath.Dir(path))
	cfg.Timeline.Releases.ChangelogPath = path
	events, err = Scan(cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3 release events", len(events))
	}
	if events[0].Type != "release" {
		t.Errorf("Type = %q, want release", events[0].Type)
	}
	if events[0].SourceRef != "release:1.2.0" {
		t.Errorf("SourceRef = %q, want release:1.2.0", events[0].SourceRef)
	}
	if events[0].Version != "1.2.0" {
		t.Errorf("Version = %q, want 1.2.0", events[0].Version)
	}
}
