package timeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func scanConfig(dir string) *config.Config {
	return &config.Config{
		Version: 1,
		Project: config.Project{Name: "p"},
		Service: config.Service{
			Name:       "payments",
			Repository: config.Repository{Provider: "github", URL: "https://github.com/example/payments"},
		},
		Timeline: &config.TimelineRef{
			Decisions: config.TimelineScanRef{Paths: []string{filepath.Join(dir, "adr", "*.md")}},
			Incidents: config.TimelineScanRef{Paths: []string{filepath.Join(dir, "pm", "*.md")}},
		},
	}
}

func TestScan_DecisionWithFrontMatter(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := writeFile(t, filepath.Join(dir, "adr"), "0007-use-postgres.md", `---
title: Use Postgres for the ledger
date: 2026-01-02
status: Accepted
adr: "7"
summary: Postgres gives us transactions the ledger needs.
---

# Something else entirely
`)

	events, err := Scan(scanConfig(dir))
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	event := events[0]
	if event.Type != "decision" {
		t.Errorf("Type = %q, want decision", event.Type)
	}
	if event.Title != "Use Postgres for the ledger" {
		t.Errorf("Title = %q, want the front matter title to beat the heading", event.Title)
	}
	if event.ADRNumber != "7" {
		t.Errorf("ADRNumber = %q, want 7", event.ADRNumber)
	}
	if event.DecisionStatus != "accepted" {
		t.Errorf("DecisionStatus = %q, want accepted (lowercased)", event.DecisionStatus)
	}
	if event.Summary != "Postgres gives us transactions the ledger needs." {
		t.Errorf("Summary = %q", event.Summary)
	}
	if event.EventDate.Format("2006-01-02") != "2026-01-02" {
		t.Errorf("EventDate = %v, want 2026-01-02", event.EventDate)
	}
	if event.SourceRef != "adr:"+filepath.ToSlash(path) {
		t.Errorf("SourceRef = %q, want an adr: path ref", event.SourceRef)
	}
	if !strings.HasPrefix(event.SourceURL, "https://github.com/example/payments/blob/") {
		t.Errorf("SourceURL = %q, want a github blob URL", event.SourceURL)
	}
}

func TestScan_DecisionFromHeadingsOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "adr"), "0012-split-the-monolith.md", `# Split the monolith

## Status

Proposed

## Context

The deploy pipeline takes 40 minutes.
Every team waits on every other team.

## Decision

Extract billing first.
`)

	cfg := scanConfig(dir)
	cfg.Timeline.Decisions.Paths = []string{filepath.Join(dir, "adr", "*.md")}

	events, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	event := events[0]
	if event.Title != "Split the monolith" {
		t.Errorf("Title = %q, want the first # heading", event.Title)
	}
	if event.ADRNumber != "0012" {
		t.Errorf("ADRNumber = %q, want the filename digits 0012", event.ADRNumber)
	}
	if event.DecisionStatus != "proposed" {
		t.Errorf("DecisionStatus = %q, want proposed from the ## Status section", event.DecisionStatus)
	}
	want := "The deploy pipeline takes 40 minutes. Every team waits on every other team."
	if event.Summary != want {
		t.Errorf("Summary = %q, want the first paragraph under ## Context: %q", event.Summary, want)
	}
}

func TestScan_DecisionWithBadStatusIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "adr"), "0001-x.md", `---
title: X
date: 2026-01-02
status: bikeshedding
---
`)

	_, err := Scan(scanConfig(dir))
	if err == nil || !strings.Contains(err.Error(), "must be one of: proposed, accepted, superseded, deprecated") {
		t.Fatalf("Scan() error = %v, want a decision status error naming the file", err)
	}
}

func TestScan_DecisionWithoutTitleIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "adr"), "0001-x.md", "## Status\n\nAccepted\n")

	_, err := Scan(scanConfig(dir))
	if err == nil || !strings.Contains(err.Error(), "no title in front matter and no '# ' heading") {
		t.Fatalf("Scan() error = %v, want a missing title error", err)
	}
}

func TestScan_IncidentDateFromFilename(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "pm"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := writeFile(t, filepath.Join(dir, "pm"), "2026-03-14-checkout-outage.md", `# Checkout outage

## Summary

The connection pool was exhausted for 22 minutes.
`)

	cfg := scanConfig(dir)
	cfg.Timeline.Decisions.Paths = nil

	events, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	event := events[0]
	if event.Type != "incident" {
		t.Errorf("Type = %q, want incident", event.Type)
	}
	if event.Title != "Checkout outage" {
		t.Errorf("Title = %q, want Checkout outage", event.Title)
	}
	if event.EventDate.Format("2006-01-02") != "2026-03-14" {
		t.Errorf("EventDate = %v, want the 2026-03-14 from the filename", event.EventDate)
	}
	if event.Summary != "The connection pool was exhausted for 22 minutes." {
		t.Errorf("Summary = %q", event.Summary)
	}
	if event.SourceRef != "incident:"+filepath.ToSlash(path) {
		t.Errorf("SourceRef = %q, want an incident: path ref", event.SourceRef)
	}
	if event.ADRNumber != "" || event.DecisionStatus != "" {
		t.Errorf("incidents must carry no adrNumber/decisionStatus, got %q/%q", event.ADRNumber, event.DecisionStatus)
	}
}

func TestScan_NilTimelineScansNothing(t *testing.T) {
	events, err := Scan(&config.Config{Version: 1})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if events != nil {
		t.Fatalf("events = %v, want nil when timeline is unset", events)
	}
}

func TestScan_SourceRefIsStableAcrossRetitling(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "adr"), "0001-x.md", "---\ntitle: First name\ndate: 2026-01-02\n---\n")
	before, err := Scan(scanConfig(dir))
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "adr"), "0001-x.md", "---\ntitle: Renamed entirely\ndate: 2026-01-02\n---\n")
	after, err := Scan(scanConfig(dir))
	if err != nil {
		t.Fatal(err)
	}

	if before[0].SourceRef != after[0].SourceRef {
		t.Fatalf("SourceRef changed from %q to %q after a retitle; the event would duplicate", before[0].SourceRef, after[0].SourceRef)
	}
	if after[0].Title != "Renamed entirely" {
		t.Fatalf("Title = %q, want the new title", after[0].Title)
	}
}
