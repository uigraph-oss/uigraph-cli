package timeline

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/git"
)

type Event struct {
	SourceRef      string    `json:"sourceRef"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary,omitempty"`
	EventDate      time.Time `json:"eventDate"`
	Version        string    `json:"version,omitempty"`
	ADRNumber      string    `json:"adrNumber,omitempty"`
	DecisionStatus string    `json:"decisionStatus,omitempty"`
	SourceLabel    string    `json:"sourceLabel,omitempty"`
	SourceURL      string    `json:"sourceUrl,omitempty"`
}

var validDecisionStatus = map[string]bool{
	"proposed":   true,
	"accepted":   true,
	"superseded": true,
	"deprecated": true,
}

type frontMatter struct {
	Title   string `yaml:"title"`
	Date    string `yaml:"date"`
	Status  string `yaml:"status"`
	ADR     string `yaml:"adr"`
	Summary string `yaml:"summary"`
}

var (
	frontMatterRe   = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n?`)
	headingRe       = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	leadingDigitsRe = regexp.MustCompile(`^(\d+)`)
	filenameDateRe  = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	changelogRe     = regexp.MustCompile(`(?m)^##\s+\[?v?([0-9][^\]\s]*)\]?\s*[-–—]?\s*(\d{4}-\d{2}-\d{2})?\s*$`)
)

// Scan walks the configured decision, incident and changelog sources and turns
// each one into an upsertable event. It is called on every `uigraph-cli sync`, so
// every sourceRef is derived from the file path or version rather than the
// title — renaming a heading updates the event instead of duplicating it.
func Scan(cfg *config.Config) ([]Event, error) {
	if cfg.Timeline == nil {
		return nil, nil
	}

	events := []Event{}
	branch := git.CaptureMetadata().Branch
	if branch == "" {
		branch = "main"
	}

	decisionPaths, err := expandGlobs(cfg.Timeline.Decisions.Paths)
	if err != nil {
		return nil, err
	}
	for _, path := range decisionPaths {
		event, err := parseDecision(path, cfg, branch)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	incidentPaths, err := expandGlobs(cfg.Timeline.Incidents.Paths)
	if err != nil {
		return nil, err
	}
	for _, path := range incidentPaths {
		event, err := parseIncident(path, cfg, branch)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if cfg.Timeline.Releases.ChangelogPath != "" {
		releases, err := ParseChangelog(cfg.Timeline.Releases.ChangelogPath)
		if err != nil {
			return nil, err
		}
		for _, release := range releases {
			events = append(events, Event{
				SourceRef:   "release:" + release.Version,
				Type:        "release",
				Title:       release.Version,
				Summary:     release.Notes,
				EventDate:   release.Date,
				Version:     release.Version,
				SourceLabel: filepath.ToSlash(cfg.Timeline.Releases.ChangelogPath),
				SourceURL:   blobURL(cfg, branch, cfg.Timeline.Releases.ChangelogPath),
			})
		}
	}

	return events, nil
}

func expandGlobs(patterns []string) ([]string, error) {
	seen := map[string]bool{}
	paths := []string{}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid timeline path pattern %q: %w", pattern, err)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("failed to stat %q: %w", match, err)
			}
			if info.IsDir() {
				continue
			}
			if seen[match] {
				continue
			}
			seen[match] = true
			paths = append(paths, match)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func parseDecision(path string, cfg *config.Config, branch string) (Event, error) {
	body, fm, err := readMarkdown(path)
	if err != nil {
		return Event{}, err
	}

	title := fm.Title
	if title == "" {
		title = firstHeading(body)
	}
	if title == "" {
		return Event{}, fmt.Errorf("%s: no title in front matter and no '# ' heading", path)
	}

	adrNumber := fm.ADR
	if adrNumber == "" {
		adrNumber = leadingDigits(filepath.Base(path))
	}

	status := fm.Status
	if status == "" {
		status = firstLineUnder(body, "Status")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && !validDecisionStatus[status] {
		return Event{}, fmt.Errorf("%s: decision status %q must be one of: proposed, accepted, superseded, deprecated", path, status)
	}

	summary := fm.Summary
	if summary == "" {
		summary = firstParagraphUnder(body, "Context")
	}
	if summary == "" {
		summary = firstParagraphUnder(body, "Decision")
	}

	eventDate, err := resolveDate(fm.Date, path, false)
	if err != nil {
		return Event{}, err
	}

	return Event{
		SourceRef:      "adr:" + filepath.ToSlash(path),
		Type:           "decision",
		Title:          title,
		Summary:        summary,
		EventDate:      eventDate,
		ADRNumber:      adrNumber,
		DecisionStatus: status,
		SourceLabel:    filepath.ToSlash(path),
		SourceURL:      blobURL(cfg, branch, path),
	}, nil
}

func parseIncident(path string, cfg *config.Config, branch string) (Event, error) {
	body, fm, err := readMarkdown(path)
	if err != nil {
		return Event{}, err
	}

	title := fm.Title
	if title == "" {
		title = firstHeading(body)
	}
	if title == "" {
		return Event{}, fmt.Errorf("%s: no title in front matter and no '# ' heading", path)
	}

	summary := fm.Summary
	if summary == "" {
		summary = firstParagraphUnder(body, "Summary")
	}
	if summary == "" {
		summary = firstParagraphUnder(body, "Impact")
	}

	eventDate, err := resolveDate(fm.Date, path, true)
	if err != nil {
		return Event{}, err
	}

	return Event{
		SourceRef:   "incident:" + filepath.ToSlash(path),
		Type:        "incident",
		Title:       title,
		Summary:     summary,
		EventDate:   eventDate,
		SourceLabel: filepath.ToSlash(path),
		SourceURL:   blobURL(cfg, branch, path),
	}, nil
}

func readMarkdown(path string) (string, frontMatter, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", frontMatter{}, fmt.Errorf("failed to read %q: %w", path, err)
	}
	content := string(raw)

	match := frontMatterRe.FindStringSubmatch(content)
	if match == nil {
		return content, frontMatter{}, nil
	}

	var fm frontMatter
	if err := yaml.Unmarshal([]byte(match[1]), &fm); err != nil {
		return "", frontMatter{}, fmt.Errorf("failed to parse front matter in %q: %w", path, err)
	}
	return content[len(match[0]):], fm, nil
}

// resolveDate walks the date sources in priority order: front matter, then a
// YYYY-MM-DD in the filename when allowed, then the commit that added the file,
// and finally the file's mtime for a file that is not committed yet. A
// malformed explicit date is an error rather than a fall-through, so a typo in
// front matter is reported instead of silently becoming the commit date.
func resolveDate(fmDate, path string, allowFilenameDate bool) (time.Time, error) {
	if fmDate != "" {
		parsed, err := parseDate(fmDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("%s: %w", path, err)
		}
		return parsed, nil
	}

	if allowFilenameDate {
		if match := filenameDateRe.FindString(filepath.Base(path)); match != "" {
			return parseDate(match)
		}
	}

	if addedAt, err := git.FileAddedAt(path); err == nil {
		return addedAt, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: no date in front matter, git or file metadata: %w", path, err)
	}
	return info.ModTime(), nil
}

func parseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("date %q is not YYYY-MM-DD or RFC3339", raw)
}

func firstHeading(body string) string {
	match := headingRe.FindStringSubmatch(body)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func leadingDigits(name string) string {
	match := leadingDigitsRe.FindStringSubmatch(name)
	if match == nil {
		return ""
	}
	return match[1]
}

// sectionBody returns everything under a `## <name>` heading up to the next
// heading of the same or higher level.
func sectionBody(body, name string) string {
	lines := strings.Split(body, "\n")
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if strings.EqualFold(heading, name) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "#") {
			return strings.Join(lines[start:i], "\n")
		}
	}
	return strings.Join(lines[start:], "\n")
}

func firstLineUnder(body, name string) string {
	for _, line := range strings.Split(sectionBody(body, name), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstParagraphUnder(body, name string) string {
	paragraph := []string{}
	for _, line := range strings.Split(sectionBody(body, name), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(paragraph) > 0 {
			break
		}
		if trimmed == "" {
			continue
		}
		paragraph = append(paragraph, trimmed)
	}
	return strings.Join(paragraph, " ")
}

// blobURL builds a browsable link to a repo file. Only github and gitlab have a
// known blob layout, so anything else gets no link rather than a guessed one.
func blobURL(cfg *config.Config, branch, path string) string {
	provider := cfg.Service.Repository.Provider
	base := strings.TrimSuffix(strings.TrimSuffix(cfg.Service.Repository.URL, "/"), ".git")
	if base == "" {
		return ""
	}
	if provider == "github" {
		return fmt.Sprintf("%s/blob/%s/%s", base, branch, filepath.ToSlash(path))
	}
	if provider == "gitlab" {
		return fmt.Sprintf("%s/-/blob/%s/%s", base, branch, filepath.ToSlash(path))
	}
	return ""
}
