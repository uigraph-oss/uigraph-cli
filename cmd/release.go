package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
	"github.com/uigraph-oss/uigraph-cli/pkg/git"
	"github.com/uigraph-oss/uigraph-cli/pkg/timeline"
)

const maxReleaseCommits = 50

var (
	releaseVersion   string
	releaseDate      string
	releaseNotes     string
	releaseNotesFile string
	releaseChangelog string
	releaseTitle     string
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Record a release on the service timeline",
	Long: `Records a single release event at the moment a tag is created, for pipelines that cut releases as a CI/CD step.
Run this from a tag-triggered job; it resolves the version, release notes and commit range from git and posts one timeline event.
Requires the UIGRAPH_TOKEN environment variable.`,
	RunE: runRelease,
}

func init() {
	releaseCmd.Flags().StringVar(&releaseVersion, "version", "", "Release version (defaults to the tag at HEAD)")
	releaseCmd.Flags().StringVar(&releaseDate, "date", "", "Release date, YYYY-MM-DD or RFC3339 (defaults to the tag date)")
	releaseCmd.Flags().StringVar(&releaseNotes, "notes", "", "Release notes text")
	releaseCmd.Flags().StringVar(&releaseNotesFile, "notes-file", "", "Path to a file holding the release notes")
	releaseCmd.Flags().StringVar(&releaseChangelog, "changelog", "CHANGELOG.md", "Changelog to pull notes from when no notes are given")
	releaseCmd.Flags().StringVar(&releaseTitle, "title", "", "Event title (defaults to the version)")
	releaseCmd.Flags().StringVar(&configPath, "config", ".uigraph.yaml", "Path to config file")
	releaseCmd.Flags().StringVar(&apiURL, "api-url", "", "Gateway API URL (defaults to UIGRAPH_GATEWAY_URL env var)")
	releaseCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the event without sending it to the gateway")
}

func runRelease(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	token := os.Getenv("UIGRAPH_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: UIGRAPH_TOKEN environment variable is required")
		os.Exit(1)
	}

	if apiURL == "" {
		apiURL = os.Getenv("UIGRAPH_GATEWAY_URL")
	}
	if apiURL == "" {
		fmt.Fprintln(os.Stderr, "Error: gateway URL is required (set --api-url or UIGRAPH_GATEWAY_URL environment variable)")
		os.Exit(1)
	}

	fmt.Printf("📦 Loading config from: %s\n", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
		os.Exit(1)
	}
	if cfg.Service.Name == "" {
		fmt.Fprintln(os.Stderr, "Error: service.name is required to record a release")
		os.Exit(1)
	}

	version := releaseVersion
	if version == "" {
		tag, err := git.TagAtHead()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: no --version given and HEAD is not tagged; pass --version explicitly")
			os.Exit(1)
		}
		version = tag
	}
	fmt.Printf("🏷️  Release: %s\n", version)

	eventDate, err := resolveReleaseDate(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	commits, commitRange := resolveReleaseCommits(version)

	notes, source, err := resolveReleaseNotes(version, commits)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📝 Notes from: %s\n", source)

	summary := notes
	if len(commits) > 0 && source != "commit subjects" {
		summary = strings.TrimSpace(summary + "\n\n" + formatCommits(commits))
	}
	if commitRange != "" {
		fmt.Printf("🔀 Commit range: %s (%d %s)\n", commitRange, len(commits), pluralize(len(commits), "commit", "commits"))
	}

	title := releaseTitle
	if title == "" {
		title = version
	}

	// The changelog scanner keys releases on the bare version, so strip a
	// leading "v" here too — a repo that both tags through CI and keeps a
	// CHANGELOG updates one event instead of recording two.
	event := timeline.Event{
		SourceRef: "release:" + strings.TrimPrefix(version, "v"),
		Type:      "release",
		Title:     title,
		Summary:   summary,
		EventDate: eventDate,
		Version:   version,
		SourceURL: releaseURL(cfg, version),
	}

	gitMeta := git.CaptureMetadata()
	req := gateway.TimelineSyncRequest{
		ServiceName: cfg.Service.Name,
		CommitHash:  gitMeta.CommitHash,
		Events:      []timeline.Event{event},
	}

	if dryRun {
		fmt.Println("\n=== DRY RUN: Release Event ===")
		fmt.Printf("  Service:   %s\n", cfg.Service.Name)
		fmt.Printf("  SourceRef: %s\n", event.SourceRef)
		fmt.Printf("  Title:     %s\n", event.Title)
		fmt.Printf("  Date:      %s\n", event.EventDate.Format(time.RFC3339))
		fmt.Printf("  SourceURL: %s\n", event.SourceURL)
		fmt.Printf("  Summary:\n%s\n", indent(event.Summary, "    "))
		fmt.Println("\n(Dry run - no data sent to gateway)")
		return nil
	}

	client := gateway.NewClient(apiURL, token)
	resp, err := client.SyncTimeline(ctx, req)
	if err != nil {
		exitGatewayErrorErr(fmt.Sprintf("record release %q", version), err)
	}
	if resp.Created == 1 {
		fmt.Printf("✓ Release recorded: %s\n", version)
	}
	if resp.Created == 0 {
		fmt.Printf("✓ Release updated: %s\n", version)
	}

	return nil
}

func resolveReleaseDate(version string) (time.Time, error) {
	if releaseDate != "" {
		if parsed, err := time.Parse(time.RFC3339, releaseDate); err == nil {
			return parsed, nil
		}
		parsed, err := time.Parse("2006-01-02", releaseDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("--date %q is not YYYY-MM-DD or RFC3339", releaseDate)
		}
		return parsed, nil
	}
	if tagged, err := git.TagDate(version); err == nil {
		return tagged, nil
	}
	return time.Now().UTC(), nil
}

// resolveReleaseCommits lists what shipped in this release. A missing previous
// tag means this is the first one, so the range is every commit up to it.
func resolveReleaseCommits(version string) ([]string, string) {
	previous, err := git.PreviousTag(version)
	if err != nil {
		previous = ""
	}

	commits, err := git.CommitsBetween(previous, version)
	if err != nil {
		return nil, ""
	}
	if previous == "" {
		return commits, version
	}
	return commits, previous + ".." + version
}

// resolveReleaseNotes walks the note sources in priority order, stopping at the
// first one that yields text, and reports which one won.
func resolveReleaseNotes(version string, commits []string) (string, string, error) {
	if releaseNotes != "" {
		return releaseNotes, "--notes", nil
	}

	if releaseNotesFile != "" {
		raw, err := os.ReadFile(releaseNotesFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to read --notes-file %q: %w", releaseNotesFile, err)
		}
		return strings.TrimSpace(string(raw)), "--notes-file", nil
	}

	if body, err := git.AnnotatedTagBody(version); err == nil && body != "" {
		return body, "annotated tag", nil
	}

	if _, err := os.Stat(releaseChangelog); err == nil {
		notes, err := timeline.ChangelogNotes(releaseChangelog, version)
		if err != nil {
			return "", "", err
		}
		if notes != "" {
			return notes, releaseChangelog, nil
		}
	}

	if len(commits) > 0 {
		return formatCommits(commits), "commit subjects", nil
	}

	return "", "no source (empty notes)", nil
}

func formatCommits(commits []string) string {
	lines := []string{fmt.Sprintf("Commits (%d):", len(commits))}
	for i, commit := range commits {
		if i == maxReleaseCommits {
			lines = append(lines, fmt.Sprintf("… and %d more", len(commits)-maxReleaseCommits))
			break
		}
		lines = append(lines, "- "+commit)
	}
	return strings.Join(lines, "\n")
}

// releaseURL links to the provider's release page for the tag. Only github and
// gitlab have a known layout, so anything else gets no link.
func releaseURL(cfg *config.Config, version string) string {
	base := strings.TrimSuffix(strings.TrimSuffix(cfg.Service.Repository.URL, "/"), ".git")
	if base == "" {
		return ""
	}
	if cfg.Service.Repository.Provider == "github" {
		return fmt.Sprintf("%s/releases/tag/%s", base, version)
	}
	if cfg.Service.Repository.Provider == "gitlab" {
		return fmt.Sprintf("%s/-/releases/%s", base, version)
	}
	return ""
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
