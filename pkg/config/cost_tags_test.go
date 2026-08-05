package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validCostTagsTestConfig() *Config {
	return &Config{
		Version: 1,
		Project: Project{Name: "test-project"},
		Service: Service{
			Name:        "payments",
			Category:    "backend",
			Description: "handles payments",
			Repository:  Repository{Provider: "github", URL: "https://github.com/example/payments"},
			Ownership:   Ownership{Team: "payments-team"},
		},
		CostTags: []CostTag{
			{Key: "team", Value: "checkout"},
			{Key: "Service", Value: "booking-api"},
		},
	}
}

func TestConfigValidate_CostTags(t *testing.T) {
	cfg := validCostTagsTestConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestConfigValidate_CostTagsEmptyClearsRules(t *testing.T) {
	cfg := validCostTagsTestConfig()
	cfg.CostTags = []CostTag{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	if cfg.CostTags == nil {
		t.Fatal("an explicit empty costTags must stay non-nil so the sync step still runs")
	}
}

func TestConfigValidate_CostTagMissingKey(t *testing.T) {
	cfg := validCostTagsTestConfig()
	cfg.CostTags[0].Key = ""
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "costTags[0].key is required") {
		t.Fatalf("Validate() error = %v, want costTags[0].key is required", err)
	}
}

func TestConfigValidate_CostTagMissingValue(t *testing.T) {
	cfg := validCostTagsTestConfig()
	cfg.CostTags[1].Value = ""
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "costTags[1].value is required") {
		t.Fatalf("Validate() error = %v, want costTags[1].value is required", err)
	}
}

func TestConfigValidate_CostTagDuplicatePair(t *testing.T) {
	cfg := validCostTagsTestConfig()
	cfg.CostTags[1] = cfg.CostTags[0]
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate key/value pair") {
		t.Fatalf("Validate() error = %v, want duplicate key/value pair", err)
	}
}

func TestConfigValidate_CostTagsRequireService(t *testing.T) {
	cfg := validCostTagsTestConfig()
	cfg.Service = Service{}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "service is required to sync costTags") {
		t.Fatalf("Validate() error = %v, want service is required to sync costTags", err)
	}
}

func TestConfigLoad_CostTagsAbsentIsNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".uigraph.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nproject:\n  name: p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CostTags != nil {
		t.Fatalf("CostTags = %v, want nil so the CLI leaves existing rules alone", cfg.CostTags)
	}
}

func TestConfigLoad_CostTagsEmptyListIsNotNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".uigraph.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nproject:\n  name: p\ncostTags: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CostTags == nil {
		t.Fatal("CostTags = nil, want an empty non-nil slice so the CLI clears every rule")
	}
}
